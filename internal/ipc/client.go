package ipc

import (
	"encoding/json"
	"fmt"
	"net"
)

type Client struct {
	path string
}

func NewClient(path string) (*Client, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("connect to daemon: %w", err)
	}
	conn.Close()
	return &Client{path: path}, nil
}

func (c *Client) Close() {}

func (c *Client) Send(req Request) (Response, error) {
	conn, err := net.Dial("unix", c.path)
	if err != nil {
		return Response{}, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, fmt.Errorf("encode request: %w", err)
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("decode response: %w", err)
	}
	return resp, nil
}
