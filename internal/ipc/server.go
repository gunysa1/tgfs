package ipc

import (
	"context"
	"encoding/json"
	"net"
	"os"
)

type HandlerFunc func(req Request) Response

type Server struct {
	path    string
	handler HandlerFunc
}

func NewServer(path string, handler HandlerFunc) *Server {
	return &Server{path: path, handler: handler}
}

func (s *Server) Serve(ctx context.Context) error {
	os.Remove(s.path)
	l, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		l.Close()
	}()
	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		return
	}
	resp := s.handler(req)
	json.NewEncoder(conn).Encode(resp)
}
