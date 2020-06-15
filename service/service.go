package service

import (
	"context"
	"errors"
	"log"
	"registry"
	"registry/etcd"
	"time"
)

type defaultRegistry struct {
	opts registry.Options
	name string
	address []string
	reg registry.Registry
}

var errCh chan error

func NewDefaultReg(opts ...registry.Option) *defaultRegistry {
	var options registry.Options
	for _, o := range opts {
		o(&options)
	}

	// the registry address
	addrs := options.Addrs
	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:8000"}
	}

	if options.Context == nil {
		options.Context = context.TODO()
	}

	// extract the client from the context, fallback to grpc

	// service name. TODO: accept option
	name := "default"

	return &defaultRegistry{
		opts:    options,
		name:    name,
		address: addrs,
		reg:  etcd.NewRegistry(opts...),
	}
}

// 注册监听下线续租为一体
func (dReg *defaultRegistry) Start(srv *registry.Service, opts ...registry.Option) error {
	if err := dReg.Init(opts...); err != nil {
		return err
	}
	errCh := make(chan error, 1)
	go dReg.watch()
	go dReg.keep(srv)
	return <-errCh
}

func (dReg *defaultRegistry) keep(srv *registry.Service) {
	t := time.NewTicker(5 * time.Second)
	for {
		<-t.C
		err := dReg.Register(srv, registry.RegisterTTL(30 * time.Second))
		if err != nil {
			errCh <- err
		}
	}
}

func (dReg *defaultRegistry) watch() {
	watcher, err := dReg.Watch()
	if err != nil {
		errCh <- err
		return
	}
	for {
		result, err := watcher.Next()
		if err != nil {
			errCh <- err
		}
		log.Printf("[注册中心事件]: %s, 服务节点: %s-%s, 地址: %s", result.Action, result.Service.Name, result.Service.Nodes[0].Id, result.Service.Nodes[0].Address)
	}
}

func (dReg *defaultRegistry) Init(opts ...registry.Option) error {
	for _, o := range opts {
		o(&dReg.opts)
	}
	if len(dReg.opts.Addrs) > 0 {
		dReg.address = dReg.opts.Addrs
	}

	dReg.reg = etcd.NewRegistry(opts...)
	if dReg.reg == nil {
		return errors.New("初始化 registry 失败")
	}
	return nil
}

func (dReg *defaultRegistry) Options() registry.Options {
	return dReg.reg.Options()
}

func (dReg *defaultRegistry) Register(srv *registry.Service, opts ...registry.RegisterOption) error {
	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.TODO()
	}

	// register the service
	err := dReg.reg.Register(srv, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (dReg *defaultRegistry) Deregister(srv *registry.Service, opts ...registry.DeregisterOption) error {
	var options registry.DeregisterOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.TODO()
	}

	// deregister the service
	err := dReg.reg.Deregister(srv, opts...)
	if err != nil {
		return err
	}
	return nil
}

func (dReg *defaultRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	var options registry.GetOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.TODO()
	}

	return dReg.reg.GetService(name, opts...)
}

func (dReg *defaultRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	var options registry.ListOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.TODO()
	}

	return dReg.reg.ListServices(opts...)
}

func (dReg *defaultRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var options registry.WatchOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.TODO()
	}
	return dReg.reg.Watch(opts...)
}

func (dReg *defaultRegistry) String() string {
	return "default register service by" + dReg.reg.String()
}

