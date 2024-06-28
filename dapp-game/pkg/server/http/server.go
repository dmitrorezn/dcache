package httpserver

import "github.com/labstack/echo/v4"

type Server struct {
	s *echo.Echo
}

func New() *Server {
	return &Server{
		s: echo.New(),
	}
}

func (s *Server) Register(route string, handler echo.HandlerFunc) {
	s.s.Any(route, handler)
}

func (s *Server) Run(addr string) error {
	return s.s.Start(addr)
}
