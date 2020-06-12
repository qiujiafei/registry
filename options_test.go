package registry

import (
	"context"
	"testing"
	"time"
)

func TestAddrs(t *testing.T) {
	var expected = "127.0.0.1"
	opt := &Options{}

	addFunc := Addrs("127.0.0.1")
	addFunc(opt)

	if  opt.Addrs[0] != expected {
		t.Errorf("Addrs actual %s, expected %s", opt.Addrs[0], expected)
	}
}

func TestTimeout(t *testing.T) {
	var (
		opt = new(Options)
		in = time.Second
		expected = time.Second
	)

	addFunc := Timeout(in)
	addFunc(opt)

	if opt.Timeout != expected {
		t.Errorf("Timeout actual %d, expected %s", opt.Timeout, expected)
	}
}

func TestRegisterTTL(t *testing.T) {
	var (
		opt = new(RegisterOptions)
		in = time.Second
		expected = time.Second
	)

	addFunc := RegisterTTL(in)
	addFunc(opt)

	if opt.TTL != expected {
		t.Errorf("TTL actual %d, expected %s", opt.TTL, expected)
	}
}

func TestRegisterContext(t *testing.T) {
	var (
		opt = new(RegisterOptions)
		in = context.Background()
		expected = context.Background()
	)

	addFunc := RegisterContext(in)
	addFunc(opt)

	if opt.Context!= expected {
		t.Errorf("Context actual %d, expected %s", opt.Context, expected)
	}
}

func TestDeregisterContext(t *testing.T) {
	var (
		opt = new(DeregisterOptions)
		in = context.Background()
		expected = context.Background()
	)

	addFunc := DeregisterContext(in)
	addFunc(opt)

	if opt.Context!= expected {
		t.Errorf("Context actual %d, expected %s", opt.Context, expected)
	}
}
