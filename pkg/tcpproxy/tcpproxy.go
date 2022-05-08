package tcpproxy

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/fly-hiring/platform-challenge/pkg/config"
	"github.com/fly-hiring/platform-challenge/pkg/listener"
	"github.com/fly-hiring/platform-challenge/pkg/listener/ebpf"
	"github.com/fly-hiring/platform-challenge/pkg/listener/native"

	"github.com/ssoroka/slice"
)

type Router interface {
	UpdateConfig(upstreams map[string][]string, portMapping map[int]string)
	Handle(context.Context, net.Conn)
}

type TCPProxy struct {
	sync.RWMutex

	listener listener.Interface
	router   Router

	config config.Config
}

func New(ctx context.Context, cfgStore *config.ConfigStore, ebpfMode bool) (*TCPProxy, error) {
	p := &TCPProxy{}

	cfg, err := cfgStore.Read()
	if err != nil {
		return nil, err
	}

	h := &naiveRouter{}
	p.router = h

	if ebpfMode {
		l, err := ebpf.NewListener(h.Handle)
		if err != nil {
			return nil, err
		}

		p.listener = l
	} else {
		l, err := native.NewListener(h.Handle)
		if err != nil {
			return nil, err
		}

		p.listener = l
	}

	go func() {
		// watch for changes to the config
		ch, err := cfgStore.StartWatcher()
		if err != nil {
			log.Fatalln(err)
		}
		defer cfgStore.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case cfg := <-ch:
				p.UpdateConfig(cfg)
			}
		}
	}()

	p.UpdateConfig(cfg)

	return p, nil
}

func (p *TCPProxy) Listen(ctx context.Context) error {
	p.RLock()
	defer p.RUnlock()

	return p.listener.Listen(ctx)
}

func (p *TCPProxy) Close() error {
	p.Lock()
	defer p.Unlock()
	return p.listener.Close()
}

func (p *TCPProxy) UpdateConfig(cfg config.Config) {
	p.Lock()
	defer p.Unlock()

	exitingPorts := p.listener.GetPorts()

	allNewPorts := []int{}
	newUpstreams := map[string][]string{}
	portMapping := map[int]string{}

	for _, app := range cfg.Apps {
		for _, port := range app.Ports {
			allNewPorts = append(allNewPorts, port)
			portMapping[port] = app.Name
		}
		newUpstreams[app.Name] = app.Targets
	}

	portsToAdd := slice.Subtract(allNewPorts, exitingPorts)
	portsToRemove := slice.Subtract(exitingPorts, allNewPorts)

	if len(portsToAdd) > 0 {
		err := p.listener.AddPorts(portsToAdd...)
		if err != nil {
			log.Println(err)
		}
	}

	if len(portsToRemove) > 0 {
		err := p.listener.RemovePorts(portsToRemove...)
		if err != nil {
			log.Println(err)
		}
	}

	p.router.UpdateConfig(newUpstreams, portMapping)
	p.config = cfg
}
