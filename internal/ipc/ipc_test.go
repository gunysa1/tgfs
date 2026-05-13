package ipc_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/gunysa1/tgfs/internal/ipc"
)

func waitForSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("unix", path)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("socket %q not ready after %v", path, timeout)
}

func TestIPCRoundtrip(t *testing.T) {
	sockPath := fmt.Sprintf("/tmp/tgfs-test-%d.sock", os.Getpid())
	os.Remove(sockPath)
	defer os.Remove(sockPath)

	handler := func(req ipc.Request) ipc.Response {
		return ipc.Response{OK: true, Data: req.Args}
	}

	srv := ipc.NewServer(sockPath, handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Serve(ctx)

	if err := waitForSocket(sockPath, 2*time.Second); err != nil {
		t.Fatal(err)
	}

	client, err := ipc.NewClient(sockPath)
	if err != nil {
		t.Fatalf("connect to server: %v", err)
	}
	defer client.Close()

	resp, err := client.Send(ipc.Request{Command: "ping", Args: map[string]string{"msg": "hello"}})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !resp.OK {
		t.Errorf("expected OK response")
	}
	if resp.Data["msg"] != "hello" {
		t.Errorf("expected echoed data, got %v", resp.Data)
	}
}

func TestIPCUnknownCommand(t *testing.T) {
	sockPath := fmt.Sprintf("/tmp/tgfs-test-unknown-%d.sock", os.Getpid())
	os.Remove(sockPath)
	defer os.Remove(sockPath)

	handler := func(req ipc.Request) ipc.Response {
		return ipc.Response{Error: "unknown command: " + req.Command}
	}

	srv := ipc.NewServer(sockPath, handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.Serve(ctx)

	if err := waitForSocket(sockPath, 2*time.Second); err != nil {
		t.Fatal(err)
	}

	client, _ := ipc.NewClient(sockPath)
	resp, err := client.Send(ipc.Request{Command: "badcmd"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.OK {
		t.Error("expected not-OK response")
	}
	if resp.Error == "" {
		t.Error("expected error message")
	}
}
