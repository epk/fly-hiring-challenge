//go:build linux
// +build linux

package sklookup

import (
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/pkg/errors"
)

const sockMapKey uint32 = 0

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc $BPF_CLANG -cflags $BPF_CFLAGS bpf proxy_dispatch.bpf.c -- -I/usr/local/include

type SkLookup struct {
	sync.Mutex

	objs      *bpfObjects
	netNsLink *link.NetNsLink
}

func New() (*SkLookup, error) {
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		return nil, errors.Wrap(err, "loading eBPF objects")

	}

	return &SkLookup{objs: &objs}, nil
}

func (s *SkLookup) Attach(listenerFD uintptr, netnsFD uintptr) error {
	s.Lock()
	defer s.Unlock()

	err := s.objs.bpfMaps.Sockets.Put(sockMapKey, uint64(listenerFD))
	if err != nil {
		return errors.Wrap(err, "updating sockets map")
	}

	s.netNsLink, err = link.AttachNetNs(int(netnsFD), s.objs.bpfPrograms.Dispatch)
	if err != nil {
		return errors.Wrap(err, "attaching eBPF program to netns")
	}

	return nil
}

func (s *SkLookup) Close() error {
	s.Lock()
	defer s.Unlock()

	err := s.netNsLink.Close()
	if err != nil {
		return errors.Wrap(err, "closing netns link")
	}

	err = s.objs.Close()
	if err != nil {
		return errors.Wrap(err, "closing eBPF objects")
	}

	return nil
}

func (s *SkLookup) AddPorts(ports ...int) error {
	s.Lock()
	defer s.Unlock()

	var val uint8 = 0

	portsToAdd := make([]uint16, len(ports))
	values := make([]uint8, len(ports))

	for i, port := range ports {
		portsToAdd[i] = uint16(port)
		values[i] = val
	}

	n, err := s.objs.bpfMaps.Destinations.BatchUpdate(portsToAdd, values, &ebpf.BatchOptions{})
	if err != nil {
		return errors.Wrap(err, "adding ports to destinations map")
	}

	if n != len(ports) {
		return errors.New("not all ports were added")
	}

	return nil
}

func (s *SkLookup) RemovePorts(ports ...int) error {
	s.Lock()
	defer s.Unlock()

	portsToRemove := make([]uint16, len(ports))
	for i, port := range ports {
		portsToRemove[i] = uint16(port)
	}

	n, err := s.objs.bpfMaps.Destinations.BatchDelete(portsToRemove, &ebpf.BatchOptions{})
	if err != nil {
		return errors.Wrap(err, "removing ports from destinations map")
	}

	if n != len(ports) {
		return errors.New("not all ports were removed")
	}

	return nil
}
