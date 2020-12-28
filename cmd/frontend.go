package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	gokitlog "github.com/go-kit/kit/log"
	"zhaokun.org/xdp-lb/pkg/controller"
	"zhaokun.org/xdp-lb/pkg/lbmap"
)

func init() {
	logger := gokitlog.NewLogfmtLogger(gokitlog.NewSyncWriter(os.Stderr))
	logger = gokitlog.With(logger, "ts", gokitlog.DefaultTimestampUTC, "caller", gokitlog.DefaultCaller)
	logger = gokitlog.With(logger, "pid", os.Getpid())
	log.SetOutput(gokitlog.NewStdlibAdapter(logger))
}

func main() {
	err := <-parseFlag().run()
	log.Fatalf("boot lbmap server error: %s", err)
}

type server struct {
	addr    string
	mapName string
}

func (s *server) run() <-chan error {
	mapper := lbmap.New()
	err := mapper.Load(s.mapName)
	if err != nil {
		c := make(chan error)
		go func() <-chan error {
			c <- fmt.Errorf("load mapper for %s error :%s", s.mapName, err)
			return c
		}()
		return c
	}
	redirectRule := controller.NewRedirectRule(mapper, s.addr)
	return redirectRule.Run()
}

func parseFlag() *server {
	addr := flag.String("address", ":9091", "Listen address of api server")
	mapName := flag.String("map", "/sys/fs/bpf/xdp/globals/servers", "name of ebpf map")
	flag.Parse()
	return &server{*addr, *mapName}
}
