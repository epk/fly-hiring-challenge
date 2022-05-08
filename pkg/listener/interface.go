package listener

import (
	"context"
)

type Interface interface {
	Listen(ctx context.Context) error
	Close() error

	AddPorts(ports ...int) error
	RemovePorts(ports ...int) error
	GetPorts() []int
}
