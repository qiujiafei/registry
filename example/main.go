package main

import (
	"log"
	"os"
	"os/signal"
	"registry"
	"registry/service"
	"syscall"
)

func main() {
	node := &registry.Node{
		Id: "node2",
		Address: "node2.com",
	}
	srv := &registry.Service{
		Name: "test",
		Version: "02",
		Nodes: []*registry.Node{node},
	}
	r := service.NewDefaultReg()
	c := make(chan os.Signal, 1)
	go func() {
		if err := r.Start(srv); err != nil {
			r.Deregister(srv)
		}
	}()
	signal.Notify(c, syscall.SIGHUP, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGUSR1)
	sig := <-c
	log.Printf("[signal]信号: %s \n", sig)
	r.Deregister(srv)
}