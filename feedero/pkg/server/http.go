package server

import (
	"context"
	"net/http"
)

type HTTPServer struct {
	srv *http.Server
}

func NewHTTP(addr string) *HTTPServer {
	return &HTTPServer{
		srv: &http.Server{
			Addr: addr,
		},
	}
}

func (s *HTTPServer) Register(h http.Handler) {
	s.srv.Handler = h
}

func (s *HTTPServer) Run() error {
	return s.srv.ListenAndServe()
}

func (s *HTTPServer) Close(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
