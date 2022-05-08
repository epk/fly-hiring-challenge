package tcpproxy

import (
	"context"
	"errors"
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ssoroka/slice"
)

type naiveRouter struct {
	sync.RWMutex

	upstreams   map[string][]string // app name -> targets
	portMapping map[int]string      // port -> app name
}

func (e *naiveRouter) Handle(ctx context.Context, src net.Conn) {
	defer goCloseConn(src)

	dst, err := e.GetUpstream(src)
	if err != nil {
		log.Println(err)
		return
	}
	defer goCloseConn(dst)

	closed := false
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer func() {
			log.Println("[DEBUG] COPY DONE", src.RemoteAddr(), "->", dst.RemoteAddr())
			cancel()
		}()

		log.Println("[DEBUG] COPY START", src.RemoteAddr(), "->", dst.RemoteAddr())

		for !closed {
			select {
			case <-ctx.Done():
				return
			default:
				src.SetReadDeadline(time.Now().Add(time.Second))
				dst.SetWriteDeadline(time.Now().Add(time.Second))

				_, err := io.Copy(dst, src)
				if isNormalTerminationError(err) {
					return
				}

				if err != nil {
					log.Println("[DEBUG] COPY ERROR", src.RemoteAddr(), "->", dst.RemoteAddr(), err)
					return
				}
			}
		}
	}()

	go func() {
		defer func() {
			log.Println("[DEBUG] COPY DONE", src.RemoteAddr(), "<-", dst.RemoteAddr())
			cancel()
		}()

		log.Println("[DEBUG] COPY START", src.RemoteAddr(), "<-", dst.RemoteAddr())

		for !closed {
			select {
			case <-ctx.Done():
				return
			default:
				dst.SetReadDeadline(time.Now().Add(time.Second))
				src.SetWriteDeadline(time.Now().Add(time.Second))

				_, err := io.Copy(src, dst)
				if isNormalTerminationError(err) {
					return
				}

				if err != nil {
					log.Println("[DEBUG] COPY ERROR", src.RemoteAddr(), "<-", dst.RemoteAddr(), err)
					return
				}
			}
		}
	}()

	<-ctx.Done()
}

func (e *naiveRouter) GetUpstream(src net.Conn) (net.Conn, error) {
	e.RLock()
	defer e.RUnlock()

	localAddr := src.LocalAddr().String()
	port := localAddr[strings.LastIndex(localAddr, ":")+1:]
	p, _ := strconv.Atoi(port)

	appName := e.portMapping[p]
	if appName == "" {
		return nil, errors.New("app not found")
	}

	upstreamsForApp := e.upstreams[appName]
	if len(upstreamsForApp) == 0 {
		return nil, errors.New("no upstreams")
	}

	return e.PickHealthyUpstream(upstreamsForApp)
}

func (e *naiveRouter) PickHealthyUpstream(upstreams []string) (net.Conn, error) {
	idx := rand.Intn(len(upstreams))
	chosen := upstreams[idx]

	dst, err := net.Dial("tcp4", chosen)
	if err != nil {
		log.Println("error dialing upstream addr", err)

		subset := slice.Subtract(upstreams, []string{chosen})
		if len(subset) == 0 {
			return nil, errors.New("no upstreams")
		}

		return e.PickHealthyUpstream(subset)
	}

	return dst, nil
}

func (e *naiveRouter) UpdateConfig(upstreams map[string][]string, portMapping map[int]string) {
	e.Lock()
	defer e.Unlock()

	e.upstreams = upstreams
	e.portMapping = portMapping
}

func goCloseConn(c net.Conn) {
	go func() {
		c.Close()
	}()
}

func isNormalTerminationError(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return true
	}
	e, ok := err.(*net.OpError)
	if ok && e.Timeout() {
		return true
	}

	for _, cause := range []string{
		"use of closed network connection",
		"broken pipe",
		"connection reset by peer",
	} {
		if strings.Contains(err.Error(), cause) {
			return true
		}
	}

	return false
}
