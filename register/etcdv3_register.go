package register

import (
	"context"
	"errors"
	"go.etcd.io/etcd/clientv3"
	"log"
	"strings"
	"sync"
	"time"
)

// EtcdRegister implements etcd registry.
type EtcdV3Register struct {
	// service address, for example, 127.0.0.1:8972
	ServiceAddress string
	// etcd addresses
	EtcdServers []string
	// base path for dpcx server, for example example/dpcx
	BasePath string
	// Registered services
	Services       map[string]service
	metasLock      sync.RWMutex
	metas          map[string]string
	Lease			int64
	UpdateInterval time.Duration

	kv      *clientv3.Client

	dying chan struct{}
	done  chan struct{}
}

type service struct{
	path string
	host string
	leaseID clientv3.LeaseID //租约ID
	//租约KeepAlive相应chan
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
}

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

func (register *EtcdV3Register) Start() error {

	if register.kv == nil {
		err := register.initEtcd()
		if err != nil {
			return err
		}
	}
	if register.Services == nil {
		register.Services = make(map[string]service)
	}

	return nil
}

func (register *EtcdV3Register) Stop() error {
	for name := range register.Services {
		_ = register.Unregister(name)
	}

	return register.kv.Close()
}

func (register *EtcdV3Register) putKeyWithLease(name string) error {
	//设置租约时间
	resp, err := register.kv.Grant(context.Background(), register.Lease)
	if err != nil {
		return err
	}

	name = "/" + register.BasePath + "/" + name + "/" + register.ServiceAddress

	//注册服务并绑定租约
	_, err = register.kv.Put(context.Background(), name, register.ServiceAddress, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}
	//设置续租 定期发送需求请求
	leaseRespChan, err := register.kv.KeepAlive(context.Background(), resp.ID)

	if err != nil {
		return err
	}

	register.Services[name] = service{
		host:          register.ServiceAddress,
		path:          name,
		leaseID:       resp.ID,
		keepAliveChan: leaseRespChan,
	}

	log.Printf("Put key:%s  val:%s  success!", name, register.ServiceAddress)
	return nil
}

// 服务注册
func (register *EtcdV3Register) Register(name string, metadata string) error {
	return register.putKeyWithLease(name)
}

// Unregister 撤销服务
func (register *EtcdV3Register) Unregister(name string) error {
	if "" == strings.TrimSpace(name) {
		return errors.New("Register service `name` can't be empty")
	}

	if register.kv == nil {
		err := register.initEtcd()
		if err != nil {
			return err
		}
	}

	var ser service
	var ok bool

	if ser, ok = register.Services[name]; !ok {
		return errors.New("service not fond for "+name)
	}

	//撤销租约
	if _, err := register.kv.Revoke(context.Background(), ser.leaseID); err != nil {
		return err
	}
	log.Println("撤销租约")

	return nil
}
