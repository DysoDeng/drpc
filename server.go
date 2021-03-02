package drpc

import (
	"context"
	"errors"
	"github.com/dysodeng/drpc/register"
	"google.golang.org/grpc"
	"log"
	"net"
	"reflect"
	"sync"
)

type Server struct {
	mu       sync.RWMutex

	reg        register.Register
	grpcServer *grpc.Server

	// AuthFunc can be used to auth.
	AuthFunc func(ctx context.Context, token string) error
}

func NewServer(reg register.Register) *Server {

	s := &Server{
		reg:        reg,
		grpcServer: grpc.NewServer(),
	}

	err := s.reg.Start()
	if err != nil {
		log.Panicln(err)
	}

	return s
}

func (s *Server) Register(service interface{}, reg interface{}, metadata string) error {

	serviceName := reflect.Indirect(reflect.ValueOf(service)).Type().Name()

	fn := reflect.ValueOf(reg)

	if fn.Kind() != reflect.Func {
		return errors.New("reg 不是一个函数")
	}
	params := make([]reflect.Value, 2)
	params[0] = reflect.ValueOf(s.grpcServer)
	params[1] = reflect.ValueOf(service)
	fn.Call(params)

	err := s.reg.Register(serviceName, metadata)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Serve(address string) {
	// 监听本地端口
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Rpc Server net.Listen err: %v", err)
	}
	log.Println(address + " Rpc Server net.Listing...")

	err = s.grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}

func (s *Server) Stop() error {
	err := s.reg.Stop()
	if err != nil {
		return err
	}
	s.grpcServer.Stop()

	return nil
}