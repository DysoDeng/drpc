package drpc

import (
	"context"
	"errors"
	"github.com/dysodeng/drpc/register"
	"google.golang.org/grpc"
	"log"
	"net"
	"reflect"
)

// Server grpc服务注册
type Server struct {
	// register 服务注册器
	register register.Register
	// grpcServer grpc
	grpcServer *grpc.Server
	// AuthFunc can be used to auth.
	AuthFunc func(ctx context.Context, token string) error
}

// NewServer 新建服务注册
func NewServer(register register.Register, opt ...grpc.ServerOption) *Server {

	server := &Server{
		register:   register,
		grpcServer: grpc.NewServer(opt...),
	}

	err := server.register.Init()
	if err != nil {
		log.Panicln(err)
	}

	return server
}

// Register 注册服务
func (s *Server) Register(service interface{}, grpcRegister interface{}, metadata string) error {

	serviceName := reflect.Indirect(reflect.ValueOf(service)).Type().Name()

	// 注册grpc服务
	fn := reflect.ValueOf(grpcRegister)
	if fn.Kind() != reflect.Func {
		return errors.New("`grpcRegister` is not a grpc registration function")
	}
	params := make([]reflect.Value, 2)
	params[0] = reflect.ValueOf(s.grpcServer)
	params[1] = reflect.ValueOf(service)
	fn.Call(params)

	// 服务发现注册
	err := s.register.Register(serviceName, metadata)
	if err != nil {
		return err
	}

	return nil
}

// Serve 启动服务监听
func (s *Server) Serve(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("rpc server net.Listen err: %v", err)
	}
	log.Printf("listening and serving grpc on: %s\n", address)

	err = s.grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}

// Stop 服务停止
func (s *Server) Stop() error {
	err := s.register.Stop()
	if err != nil {
		return err
	}
	s.grpcServer.Stop()

	log.Println("grpc server stop")

	return nil
}
