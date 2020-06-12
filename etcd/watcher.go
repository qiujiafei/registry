package etcd

import (
	"context"
	"errors"
	"github.com/coreos/etcd/clientv3"
	"registry"
	"time"
)

type etcdWatcher struct {
	stop chan bool
	w clientv3.WatchChan
	client *clientv3.Client
	timeout time.Duration
}

func newEtcdWatcher(r *etcdRegistry, timeout time.Duration, opts ...registry.WatchOption) registry.Watcher {
	var watchOpt registry.WatchOptions

	for _, o := range opts {
		o(&watchOpt)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan bool, 1)

	// 开启协程监听退出通道
	go func() {
		<-stop
		defer cancel()
	}()

	// 监听的目录前缀
	watchPath := prefix
	if len(watchOpt.Service) > 0 {
		watchPath = servicePath(watchOpt.Service) + "/"
	}
	return &etcdWatcher{
		stop: stop,
		w: r.client.Watch(ctx, watchPath, clientv3.WithPrefix(), clientv3.WithPrevKV()),
		client: r.client,
		timeout: timeout,
	}
}

func (etcdw *etcdWatcher) Next() (*registry.Result, error) {
	for watherEtcdRsp := range etcdw.w {
		if watherEtcdRsp.Err() != nil {
			return nil, watherEtcdRsp.Err()
		}
		if watherEtcdRsp.Canceled {
			return nil, errors.New("已被取消")
		}
		for _, event := range watherEtcdRsp.Events {
			service := decode(event.Kv.Value)
			var action string

			switch event.Type {
			// put 有新增和覆盖两种
			case clientv3.EventTypePut:
				if event.IsCreate() {
					action = "create"
				} else if event.IsModify() {
					action = "update"
				}
			case clientv3.EventTypeDelete:
				action = "delete"
				service = decode(event.PrevKv.Value)
			}
			// 如果没有 service 则继续监听
			if service == nil {
				continue
			}

			return &registry.Result{
				Action: action,
				Service: service,
			}, nil
		}
	}
	return nil, errors.New("watchChan 被关闭了")
}

func (etcdw *etcdWatcher) Stop() {
	// 如果接收到了 stop 通道的返回值则退出
	// 没有就主动关闭
	select {
	case <-etcdw.stop:
		return
	default:
		close(etcdw.stop)
	}
}