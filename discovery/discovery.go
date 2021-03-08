package discovery

import (
	"google.golang.org/grpc"
)

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	Conn(service interface{}) *grpc.ClientConn
	Close()
}
