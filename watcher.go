package registry

import "time"

type Watcher interface {
	Next() (*Result, error)
	Stop()
}

type Result struct {
	Action string
	Service *Service
}

type EventType int

const (
	// 注册
	Create EventType = iota
	// 下线
	Delete
	// 更新
	Update
)
// 返回的 0，1，2 看不懂， 格式化一下
func (t EventType) String() string {
	switch t {
	case Create:
		return "create"
	case Delete:
		return "delete"
	case Update:
		return "update"
	default:
		return "unknown"
	}
}

type Event struct {
	Id string
	Type EventType
	Timestamp time.Time
	Service *Service
}

