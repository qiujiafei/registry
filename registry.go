package registry

type Registry interface {
	// 初始化配置
	Init(...Option) error
	// 获取配置
	Options() Options
	// 注册
	Register(*Service, ...RegisterOption) error
	// 离线
	Deregister(*Service, ...DeregisterOption) error
	// 获取服务
	GetService(string, ...GetOption) ([]*Service, error)
	// 服务列表
	ListServices(...ListOption) ([]*Service, error)
	// 监听
	Watch(...WatchOption) (Watcher, error)
	// 格式化输出
	String() string
}

type Service struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Metadata  map[string]string `json:"metadata"`
	Nodes     []*Node           `json:"nodes"`
}

type Node struct {
	Id       string            `json:"id"`
	Address  string            `json:"address"`
	Metadata map[string]string `json:"metadata"`
}

type Option func(*Options)

type RegisterOption func(*RegisterOptions)

type DeregisterOption func(*DeregisterOptions)

type WatchOption func(*WatchOptions)

type GetOption func(*GetOptions)

type ListOption func(*ListOptions)