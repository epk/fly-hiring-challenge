package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/features"
	"github.com/cilium/ebpf/rlimit"
	"github.com/fly-hiring/platform-challenge/pkg/config"
	"github.com/fly-hiring/platform-challenge/pkg/tcpproxy"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfgStore := config.NewConfigStore("./config.json")

	var proxy *tcpproxy.TCPProxy

	if err := features.HaveProgType(ebpf.SkLookup); err == nil {
		log.Println("SK_LOOKUP is supported, using eBPF mode")
		if err := rlimit.RemoveMemlock(); err != nil {
			log.Fatal(err)
		}

		// Start TCPProxy with eBPF mode
		proxy, err = tcpproxy.New(ctx, cfgStore, true)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		log.Println("SK_LOOKUP is not supported, using native mode")

		// Start TCPProxy with native mode
		proxy, err = tcpproxy.New(ctx, cfgStore, false)
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := proxy.Listen(ctx); err != nil {
		log.Fatalln(err)
	}

	<-ctx.Done()

	if err := proxy.Close(); err != nil {
		log.Fatalln(err)
	}
}
