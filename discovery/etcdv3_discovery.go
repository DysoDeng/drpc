package discovery

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"log"
	"sync"
	"time"
)

var etcdV3DiscoveryOnce sync.Once
var etcdV3Discovery *EtcdV3Discovery

var resolverLock sync.Mutex

// EtcdV3Discovery implements etcd discovery.
type EtcdV3Discovery struct {
	kv         *clientv3.Client
	cc         resolver.ClientConn
	serverList map[string]resolver.Address //服务列表
	lock       sync.Mutex
	basePath   string
	etcdAddress []string
}

// NewEtcdV3Discovery 新建etcd服务中心连接
func NewEtcdV3Discovery(address []string, basePath string) (ServiceDiscovery, error) {
	d := &EtcdV3Discovery{
		basePath: basePath,
		etcdAddress: address,
	}

	resolverLock.Lock()
	defer func() {
		resolverLock.Unlock()
	}()
	resolver.Register(d)

	return d, nil
}

// Conn 连接服务
func (d *EtcdV3Discovery) Conn(serviceName string) *grpc.ClientConn {
	// 连接服务器
	conn, err := grpc.Dial(
		d.Scheme()+"://8.8.8.8/"+serviceName,
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, RoundRobin)),
		grpc.WithInsecure(),
	)

	if err != nil {
		log.Fatalf("grpc net.Connect err: %v", err)
	}

	return conn
}

// Build creates a new resolver for the given target.
func (d *EtcdV3Discovery) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   d.etcdAddress,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	d.cc = cc
	d.serverList = make(map[string]resolver.Address)
	prefix := "/" + target.Scheme + "/" + target.Endpoint + "/"
	// 根据前缀获取现有的key
	resp, err := cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	for _, ev := range resp.Kvs {
		d.SetServiceList(string(ev.Key), string(ev.Value))
	}
	d.cc.UpdateState(resolver.State{Addresses: d.getServices()})
	// 监视前缀，修改变更的server
	go func() {
		rch := cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				switch ev.Type {
				case mvccpb.PUT: //新增或修改
					d.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
				case mvccpb.DELETE: //删除
					d.DeleteServiceList(string(ev.Kv.Key))
				}
			}
		}
	}()

	return d, nil
}

// Scheme returns the scheme supported by this resolver.
func (d *EtcdV3Discovery) Scheme() string {
	return d.basePath
}

// ResolveNow 监视目标更新
func (d *EtcdV3Discovery) ResolveNow(rn resolver.ResolveNowOptions) {
	//log.Println("ResolveNow")
}

// Close 关闭服务
func (d *EtcdV3Discovery) Close() {
	//_ = d.kv.Close()
}

// watcher 监听前缀
func (d *EtcdV3Discovery) watcher(prefix string) {
	rch := d.kv.Watch(context.Background(), prefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //新增或修改
				d.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				d.DeleteServiceList(string(ev.Kv.Key))
			}
		}
	}
}

// SetServiceList 新增服务地址
func (d *EtcdV3Discovery) SetServiceList(key, val string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.serverList[key] = resolver.Address{Addr: val}
	d.cc.UpdateState(resolver.State{Addresses: d.getServices()})
}

// DeleteServiceList 删除服务地址
func (d *EtcdV3Discovery) DeleteServiceList(key string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	delete(d.serverList, key)
	d.cc.UpdateState(resolver.State{Addresses: d.getServices()})
}

func (d *EtcdV3Discovery) GetServiceList() map[string]resolver.Address {
	return d.serverList
}

// getServices 获取服务地址
func (d *EtcdV3Discovery) getServices() []resolver.Address {
	addrs := make([]resolver.Address, 0, len(d.serverList))

	for _, v := range d.serverList {
		addrs = append(addrs, v)
	}
	return addrs
}

func init() {

}
