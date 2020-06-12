package etcd

import (
	"fmt"
	"reflect"
	"registry"
	"testing"
	"time"
)

func TestEncodeDecode(t *testing.T) {
	srv := &registry.Service{
		Name: "User",
		Version: "south-01",
	}
	str := encode(srv)
	srv2 := decode([]byte(str))
	if !reflect.DeepEqual(srv, srv2) {
		t.Error("序列化和反序列化失败")
	}
}

func TestNodePath(t *testing.T) {
	actual := nodePath("user", "1")
	expected := "/han/user/1"
	if actual != expected {
		t.Errorf("expected %s, actual %s", expected, actual)
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry(registry.Addrs("127.0.0.1:2379"))
	if r == nil {
		t.Error("注册 etcd 失败")
	}
}

func TestOptions(t *testing.T) {
	r := NewRegistry(registry.Addrs("127.0.0.1:2379"))
	if r.Options().Addrs[0] != "127.0.0.1:2379" {
		t.Error("属性设置失败")
	}
}

func TestInit(t *testing.T) {
	timeout := 3 * time.Second
	r := NewRegistry(registry.Addrs("127.0.0.1:2379"))
	err := r.Init(registry.Timeout(timeout))
	if err != nil {
		t.Errorf("init 失败，失败原因%s", err)
	}

	if r.Options().Timeout != timeout {
		t.Errorf("timeout 实际值 %d, 正确应是 %d", r.Options().Timeout, timeout)
	}
}

func TestString(t *testing.T) {
	r := NewRegistry(registry.Addrs("127.0.0.1:2379"))
	if "etcd" != fmt.Sprint(r) {
		t.Errorf("格式化输出 %s, 应该是 %s", fmt.Sprint(r), "etcd")
	}
}
