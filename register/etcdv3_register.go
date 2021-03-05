package register

import (
	"context"
	"errors"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"log"
	"strings"
	"sync"
	"time"
)

// EtcdV3Register implements etcd registry.
type EtcdV3Register struct {
	// service address, for example, 127.0.0.1:8972
	ServiceAddress string
	// etcd addresses
	EtcdServers []string
	// base path for dpcx server, for example example/dpcx
	BasePath string
	// Registered services
	services    map[string]service
	serviceLock sync.Mutex
	// 租约时长(秒)
	Lease       int64
	metasLock   sync.RWMutex
	metas       map[string]string

	// etcd client
	kv *clientv3.Client
}

// service 注册服务类型
type service struct {
	// 服务标识key
	key string
	// 服务host
	host string
	// 租约ID
	leaseID clientv3.LeaseID
	// 租约KeepAlive
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
}

// initEtcd 初始化etcd连接
func (register *EtcdV3Register) initEtcd() error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   register.EtcdServers,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}

	register.kv = cli

	return nil
}

// Init 初始化Etcd注册
func (register *EtcdV3Register) Init() error {
	if register.kv == nil {
		err := register.initEtcd()
		if err != nil {
			return err
		}
	}
	if register.services == nil {
		register.services = make(map[string]service)
	}
	if register.metas == nil {
		register.metas = make(map[string]string)
	}

	return nil
}

// Stop 停止服务注册
func (register *EtcdV3Register) Stop() error {
	// 注销所有服务
	for name := range register.services {
		_ = register.Unregister(name)
	}

	return register.kv.Close()
}

// Register 服务注册
func (register *EtcdV3Register) Register(name string, metadata string) error {
	// 设置租约时间
	resp, err := register.kv.Grant(context.Background(), register.Lease)
	if err != nil {
		return err
	}

	name = "/" + register.BasePath + "/" + name + "/" + register.ServiceAddress

	register.serviceLock.Lock()
	defer func() {
		register.serviceLock.Unlock()
	}()

	// 注册服务并绑定租约
	_, err = register.kv.Put(context.Background(), name, register.ServiceAddress, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}

	// 设置续租 并定期发送续租请求(心跳)
	leaseRespChan, err := register.kv.KeepAlive(context.Background(), resp.ID)

	if err != nil {
		return err
	}

	if register.services == nil {
		register.services = make(map[string]service)
	}
	register.services[name] = service{
		host:          register.ServiceAddress,
		key:           name,
		leaseID:       resp.ID,
		keepAliveChan: leaseRespChan,
	}

	register.metasLock.Lock()
	if register.metas == nil {
		register.metas = make(map[string]string)
	}
	register.metas[name] = metadata
	register.metasLock.Unlock()

	log.Printf("register service: %s", name)

	return nil
}

// Unregister 注销服务
func (register *EtcdV3Register) Unregister(name string) error {
	if "" == strings.TrimSpace(name) {
		return errors.New("register service `name` can't be empty")
	}

	if register.kv == nil {
		err := register.initEtcd()
		if err != nil {
			return err
		}
	}

	var ser service
	var ok bool

	if ser, ok = register.services[name]; !ok {
		return errors.New(fmt.Sprintf("service `%s` not registered", name))
	}

	register.serviceLock.Lock()
	defer func() {
		register.serviceLock.Unlock()
	}()

	// 撤销租约
	if _, err := register.kv.Revoke(context.Background(), ser.leaseID); err != nil {
		return err
	}

	log.Printf("unregister service: %s", name)

	return nil
}
