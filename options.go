package registry

import (
	"context"
	"time"
)


// 注册中心的连接地址 ，超时时长等参数
type Options struct {
	Addrs []string
	Timeout time.Duration
	Context context.Context
}

// 服务去注册的时候过期时间等配置
type RegisterOptions struct {
	TTL time.Duration
	Context context.Context
}

type WatchOptions struct {
	Service string

	Context context.Context
}

type DeregisterOptions struct {
	Context context.Context
}

type GetOptions struct {
	Context context.Context
}

type ListOptions struct {
	Context context.Context
}

func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

func RegisterTTL(t time.Duration) RegisterOption {
	return func(o *RegisterOptions) {
		o.TTL = t
	}
}

func RegisterContext(ctx context.Context) RegisterOption {
	return func(o *RegisterOptions) {
		o.Context =  ctx
	}
}

func DeregisterContext(ctx context.Context) DeregisterOption {
	return func(o *DeregisterOptions) {
		o.Context = ctx
	}
}