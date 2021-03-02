package discovery

import "google.golang.org/grpc"

// 服务发现接口
type ServiceDiscovery interface {
	Conn(service interface{}) *grpc.ClientConn
	Close()
}
