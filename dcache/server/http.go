package server

import (
	"context"
	"net"
	"net/http"
	"time"
)

type HTTPServer struct {
	*http.Server
	ln net.Listener
}

func NewHTTP(addr string) *HTTPServer {
	return &HTTPServer{
		Server: &http.Server{
			Addr: addr,
		},
	}
}

func (s *HTTPServer) Register(h http.Handler) {
	s.Handler = h
}

func (s *HTTPServer) Addr() net.Addr {
	return s.ln.Addr()
}

func (s *HTTPServer) Run() (err error) {
	if s.ln, err = net.Listen("tcp", s.Server.Addr); err != nil {
		return err
	}

	return s.Server.Serve(s.ln)
}

func (s *HTTPServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.Server.Shutdown(ctx)
}
