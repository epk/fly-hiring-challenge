//go:build linux
// +build linux

package ebpf

import (
	"context"
	"log"
	"net"
	"sync"
	"syscall"

	"github.com/fly-hiring/platform-challenge/pkg/listener"

	"github.com/containernetworking/plugins/pkg/ns"
	sklookup "github.com/fly-hiring/platform-challenge/pkg/sk_lookup"
)

const eBPFModeAddr = ":4444"

type ebpfListener struct {
	sync.RWMutex

	ports        []int
	baseListener net.Listener

	skLookup   *sklookup.SkLookup
	handleFunc func(context.Context, net.Conn)
}

var (
	_ listener.Interface = &ebpfListener{}
)

func NewListener(handleFunc func(context.Context, net.Conn)) (*ebpfListener, error) {
	s, err := sklookup.New()
	if err != nil {
		return nil, err
	}

	return &ebpfListener{
		skLookup:   s,
		handleFunc: handleFunc,
	}, nil
}

func (e *ebpfListener) Listen(ctx context.Context) error {
	var listenerFD uintptr

	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			c.Control(func(fd uintptr) {
				listenerFD = fd
			})
			return nil
		},
	}

	lis, err := lc.Listen(ctx, "tcp", eBPFModeAddr)
	if err != nil {
		return err
	}
	e.baseListener = lis

	go func() {
		e.serveListener(e.baseListener)
	}()

	fd, err := ns.GetCurrentNS()
	if err != nil {
		return err
	}

	if err := e.skLookup.Attach(listenerFD, fd.Fd()); err != nil {
		return err
	}

	return nil
}

func (e *ebpfListener) Close() error {
	err := e.skLookup.Close()
	if err != nil {
		return err
	}

	err = e.baseListener.Close()
	if err != nil {
		return err
	}

	for _, port := range e.ports {
		log.Printf("Closed port %d\n", port)
	}

	return nil
}

func (e *ebpfListener) AddPorts(ports ...int) error {
	e.Lock()
	defer e.Unlock()

	portsToAdd := map[int]struct{}{}
	for _, port := range ports {
		portsToAdd[port] = struct{}{}
	}

	for _, port := range e.ports {
		delete(portsToAdd, port)
	}

	diff := make([]int, 0, len(portsToAdd))
	for port := range portsToAdd {
		diff = append(diff, port)
	}
	e.ports = append(e.ports, diff...)

	if err := e.skLookup.AddPorts(diff...); err != nil {
		return err
	}

	for _, port := range diff {
		log.Printf("Added port %d\n", port)
	}

	return nil
}

func (e *ebpfListener) RemovePorts(ports ...int) error {
	e.Lock()
	defer e.Unlock()

	portsToDelete := map[int]struct{}{}
	for _, port := range ports {
		portsToDelete[port] = struct{}{}
	}

	// If the port is not in e.ports, remove it from portsToDelete
	for _, port := range e.ports {
		if _, ok := portsToDelete[port]; !ok {
			delete(portsToDelete, port)
		}
	}

	diff := make([]int, 0, len(portsToDelete))
	for port := range portsToDelete {
		diff = append(diff, port)
	}

	for _, port := range diff {
		for i, p := range e.ports {
			if p == port {
				e.ports = append(e.ports[:i], e.ports[i+1:]...)
				break
			}
		}
	}

	if err := e.skLookup.RemovePorts(diff...); err != nil {
		return err
	}

	for _, port := range diff {
		log.Printf("Removed port %d\n", port)
	}

	return nil
}

func (e *ebpfListener) GetPorts() []int {
	e.RLock()
	defer e.RUnlock()

	return e.ports
}

func (e *ebpfListener) serveListener(lis net.Listener) {
	for {
		c, err := lis.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Printf("Timeout error: %s\n", err)
				continue
			}

			return
		}

		go func() {
			e.handleFunc(context.Background(), c)
		}()
	}
}
