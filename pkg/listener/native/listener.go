package native

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/fly-hiring/platform-challenge/pkg/listener"

	"go.uber.org/multierr"
)

type nativeListener struct {
	sync.RWMutex

	listeners  map[int]net.Listener
	handleFunc func(context.Context, net.Conn)
}

var (
	_ listener.Interface = &nativeListener{}
)

func NewListener(handleFunc func(context.Context, net.Conn)) (*nativeListener, error) {
	return &nativeListener{
		listeners:  make(map[int]net.Listener),
		handleFunc: handleFunc,
	}, nil
}

func (n *nativeListener) Listen(ctx context.Context) error {
	return nil
}

func (n *nativeListener) Close() error {
	multiErr := make([]error, 0)
	for port, l := range n.listeners {
		err := l.Close()
		if err != nil {
			multiErr = append(multiErr, err)
			continue
		}

		delete(n.listeners, port)
		log.Printf("Removed port %d\n", port)
	}

	return multierr.Combine(multiErr...)
}

func (n *nativeListener) AddPorts(ports ...int) error {
	n.Lock()
	defer n.Unlock()

	portsToAdd := map[int]struct{}{}
	for _, port := range ports {
		portsToAdd[port] = struct{}{}
	}

	for port := range n.listeners {
		delete(portsToAdd, port)
	}

	diff := make([]int, 0, len(portsToAdd))
	for port := range portsToAdd {
		diff = append(diff, port)
	}

	multiErr := make([]error, 0)
	for _, port := range diff {
		l, err := net.Listen("tcp", ":"+fmt.Sprint(port))
		if err != nil {
			multiErr = append(multiErr, err)
			continue
		}

		go func() {
			n.serveListener(l)
		}()

		n.listeners[port] = l
		log.Printf("Added port %d\n", port)
	}

	return multierr.Combine(multiErr...)
}

func (n *nativeListener) RemovePorts(ports ...int) error {
	n.Lock()
	defer n.Unlock()

	portsToDelete := map[int]struct{}{}
	for _, port := range ports {
		portsToDelete[port] = struct{}{}
	}

	// If the port is not in n.listeners, remove it from portsToDelete
	for port := range n.listeners {
		if _, ok := portsToDelete[port]; !ok {
			delete(portsToDelete, port)
		}
	}

	diff := make([]int, 0, len(portsToDelete))
	for port := range portsToDelete {
		diff = append(diff, port)
	}

	multiErr := make([]error, 0)
	for _, port := range diff {
		l := n.listeners[port]
		err := l.Close()
		if err != nil {
			multiErr = append(multiErr, err)
			continue
		}

		delete(n.listeners, port)
		log.Printf("Removed port %d\n", port)
	}

	return multierr.Combine(multiErr...)
}

func (e *nativeListener) GetPorts() []int {
	e.RLock()
	defer e.RUnlock()

	ports := make([]int, 0, len(e.listeners))

	for port := range e.listeners {
		ports = append(ports, port)
	}

	return ports
}

func (e *nativeListener) serveListener(lis net.Listener) {
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
