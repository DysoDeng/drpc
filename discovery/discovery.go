package discovery

import (
	"google.golang.org/grpc"
)

const (
	RoundRobin string = "round_robin"
)

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	Conn(serviceName string) *grpc.ClientConn
	Close()
}
