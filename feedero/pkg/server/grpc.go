package server

import (
	"net"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	addr string
	srv  *grpc.Server
}

func NewGRPC(addr string) *GRPCServer {
	return &GRPCServer{
		addr: addr,
		srv:  grpc.NewServer(),
	}
}

func (s *GRPCServer) Register(services ...func(registrar grpc.ServiceRegistrar)) {
	for _, svc := range services {
		svc(s.srv)
	}
}

func (s *GRPCServer) Run() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	return s.srv.Serve(ln)
}

func (s *GRPCServer) Close() {
	s.srv.GracefulStop()
}
