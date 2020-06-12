package etcd

import (
	"context"
	"encoding/json"
	"github.com/coreos/etcd/clientv3"
	"errors"
	"net"
	"path"
	"reflect"
	"registry"
	"sort"
	"strings"
	"sync"
	"time"
)

var prefix = "/han"

type etcdRegistry struct {
	client *clientv3.Client
	options registry.Options
	// 节点列表 格式为 user.01
	regNode map[string]*registry.Node
	// 租约列表 格式为 user.01
	leases map[string]clientv3.LeaseID
	sync.RWMutex
}


func NewRegistry(opts ...registry.Option) registry.Registry {

	e := &etcdRegistry{
		options: registry.Options{},
		regNode: make(map[string]*registry.Node),
		leases: make(map[string]clientv3.LeaseID),
	}
	// 去 etcd 注册， 如果发生错误则返回空
	if configureEtcd(e, opts...) != nil {
		return nil
	}
	return e
}

// 这里初始化 etcd
func configureEtcd(e *etcdRegistry, opts ...registry.Option) error {
	// 初始化 etcd 默认地址 127.0.0.1:2379
	config := clientv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	}

	// 传过来的配置保存到 e.options 结构中
	for _, o := range opts {
		o(&e.options)
	}

	// timeout 如果没有值则设置默认值
	if e.options.Timeout == 0 {
		e.options.Timeout = 5 * time.Second
	}

	// 查询传过来的 etcd 地址
	var etcdAddrs []string
	for _, address := range e.options.Addrs {
		// 空字符串跳过
		if len(address) == 0 {
			continue
		}
		// 这里检验传入的地址格式是否合法
		addr, port, err := net.SplitHostPort(address)
		// 如果没有端口则给默认端口
		// 地址和端口都没有则跳过
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "2379"
			addr = address
			etcdAddrs = append(etcdAddrs, net.JoinHostPort(addr, port))
		} else if err == nil {
			etcdAddrs = append(etcdAddrs, net.JoinHostPort(addr, port))
		}
	}

	// 如果保存地址的切片不为空，则覆盖掉初始的 127.0.0.1
	if len(etcdAddrs) > 0 {
		config.Endpoints = etcdAddrs
	}

	cli, err := clientv3.New(config)
	if err != nil {
		return err
	}
	e.client = cli
	return nil
}

//将 service 序列化成 json
func encode(s *registry.Service) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func decode(d []byte) *registry.Service {
	var s *registry.Service
	json.Unmarshal(d, &s)
	return s
}

func nodePath(s, id string) string {
	service := strings.Replace(s, "/", "-", -1)
	node := strings.Replace(id, "/", "-", -1)
	return path.Join(prefix, service, node)
}

func servicePath(s string) string {
	return path.Join(prefix, strings.Replace(s, "/", "-", -1))
}

func (e *etcdRegistry) Init(opts ...registry.Option) error {
	return configureEtcd(e, opts ...)
}

func (e *etcdRegistry) Options() registry.Options {
	return e.options
}

func (e *etcdRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("不允许注册空节点的服务")
	}

	// 外层定义一个包含所有的 global error
	var gerr error
	for _, node := range s.Nodes {
		err := e.registerNode(s, node, opts ...)
		if err != nil {
			gerr = err
		}
	}
	return gerr
}

// 根据服务和节点来注册或者续约
func (e *etcdRegistry) registerNode(s *registry.Service, node *registry.Node, opts ...registry.RegisterOption) error {
	// 先看下对应的服务节点是否有缓存
	e.RLock()
	leaseID, ok := e.leases[s.Name + node.Id]
	e.RUnlock()

	if !ok {
		ctx, cancel := context.WithTimeout(context.Background(), e.options.Timeout)
		defer cancel()
		// 缓存中没拿到， 则去 etcd 拿
		rsp, err := e.client.Get(ctx, nodePath(s.Name, node.Id), clientv3.WithSerializable())
		if err != nil {
			return err
		}
		// 循环读取 kv 数组
		for _, kv := range rsp.Kvs {
			// 如果当前的 kv 带有租约
			if kv.Lease > 0 {
				leaseID = clientv3.LeaseID(kv.Lease)
				srv := decode(kv.Value)
				if srv == nil || len(srv.Nodes) == 0 {
					// 从 etcd 拿到的 service 有问题，跳过拿下一个
					continue
				}
				// 拿到了正确的服务 租约
				// 更新缓存
				e.Lock()
				e.leases[s.Name + node.Id] = leaseID
				e.regNode[s.Name + node.Id] = node
				e.Unlock()
				// 找到了对应 service + nodeId 就不用看其他的了
				break
			}
		}
	}

	// 找到租约 id 了就去续租一下
	if leaseID > 0 {
		if _, err := e.client.KeepAliveOnce(context.TODO(), leaseID); err != nil {
			return err
		}
	}
	e.RLock()
	v, ok := e.regNode[s.Name + node.Id]
	e.RUnlock()
	// 如果缓存中存在一样的 node ，就可以直接返回
	// 证明节点是老的，只是来续约的
	if ok && reflect.DeepEqual(v, node){
		return nil
	}

	// 能走到这里说明服务是新创建的或者服务发生了改变
	// 重新注册
	service := &registry.Service{
		Name: s.Name,
		Version: s.Version,
		Metadata: s.Metadata,
		Nodes: []*registry.Node{node},	// 只有这里的 node 需要用传进来的，因为 node 改变了
	}

	return e.register(service, opts...)
}

// etcd 注册
func (e *etcdRegistry) register(service *registry.Service, opts ...registry.RegisterOption) error {
	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.options.Timeout)
	defer cancel()

	var lgr *clientv3.LeaseGrantResponse
	var err error
	// 如果参数中带有 TTL 则证明想要一个过期的功能
	// 所以去申请一个 TTL 租约
	if options.TTL.Seconds() > 0 {
		lgr, err = e.client.Grant(ctx, int64(options.TTL.Seconds()))
		if err != nil {
			return err
		}
	}

	// 有租约就带着租约把服务注册进 etcd
	// 没有租约就直接放进 etcd
	// ? 这里为什么还可以放入不带租约的服务呢
	if lgr != nil {
		_, err = e.client.Put(ctx, nodePath(service.Name, service.Nodes[0].Id), encode(service), clientv3.WithLease(lgr.ID))
	} else {
		_, err = e.client.Put(ctx, nodePath(service.Name, service.Nodes[0].Id), encode(service))
	}
	if err != nil {
		return err
	}

	// 将新的租约和新的服务放入缓存
	e.Lock()
	e.regNode[service.Name + service.Nodes[0].Id] = service.Nodes[0]
	if lgr != nil {
		e.leases[service.Name + service.Nodes[0].Id] = lgr.ID
	}
	e.Unlock()
	return nil
}

func (e *etcdRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("不允许操作空节点")
	}
	for _, node := range s.Nodes {
		// 首先删除两个缓存
		e.Lock()
		delete(e.regNode, s.Name + node.Id)
		delete(e.leases, s.Name + node.Id)
		e.Unlock()
		// 删除 etcd
		if err := e.delete(nodePath(s.Name, node.Id)); err != nil {
			return err
		}
	}
	return nil
}

// 从 etcd 中删除
func (e *etcdRegistry) delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.options.Timeout)
	defer cancel()
	_, err := e.client.Delete(ctx, key)
	if err != nil {
		return err
	}
	return nil
}

func (e *etcdRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.options.Timeout)
	defer cancel()
	rsp, err := e.client.Get(ctx, servicePath(name)+"/", clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, errors.New("service not found")
	}

	serviceMap := map[string]*registry.Service{}

	for _, kv := range rsp.Kvs {
		if sn := decode(kv.Value); sn != nil {
			// 将相同版本号的 node 拼接在一起
			s, ok := serviceMap[sn.Version]
			if !ok {
				s = &registry.Service{
					Name: sn.Name,
					Version: sn.Version,
					Metadata: sn.Metadata,
				}
				serviceMap[s.Version] = s
			}

			s.Nodes = append(s.Nodes, sn.Nodes...)
		}
	}

	// 根据 service.Version 归类好的 node
	services := make([]*registry.Service, len(serviceMap))
	for _, srv := range serviceMap {
		services = append(services, srv)
	}
	return services, nil
}

func (e *etcdRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	versions := make(map[string]*registry.Service)
	ctx, cancel := context.WithTimeout(context.Background(), e.options.Timeout)
	defer cancel()

	rsp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}
	if  len(rsp.Kvs) == 0 {
		return nil, errors.New("service not found")
	}

	for _, kv := range rsp.Kvs {
		sn := decode(kv.Value)
		if sn == nil {
			continue
		}
		v, ok := versions[sn.Name+sn.Version]
		if !ok {
			versions[sn.Name+sn.Version] = sn
			continue
		}
		v.Nodes = append(v.Nodes, sn.Nodes...)
	}

	services := make([]*registry.Service, 0, len(versions))
	for _, service := range versions {
		services = append(services, service)
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}

func (e *etcdRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newEtcdWatcher(e, e.options.Timeout, opts...), nil
}

func (e *etcdRegistry) String() string {
	return "etcd"
}