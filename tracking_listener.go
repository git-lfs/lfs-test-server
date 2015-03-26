package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
)

type TrackingListener struct {
	wg          sync.WaitGroup
	connections map[net.Conn]bool
	cm          sync.Mutex
	net.Listener
}

func NewTrackingListener(addr string) (*TrackingListener, error) {
	var listener net.Listener

	a, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	switch a.Scheme {
	case "fd":
		fd, err := strconv.Atoi(a.Host)
		if err != nil {
			return nil, err
		}

		f := os.NewFile(uintptr(fd), "trackinglistener")
		listener, err = net.FileListener(f)
		if err != nil {
			return nil, err
		}
	case "tcp", "tcp4", "tcp6":
		laddr, err := net.ResolveTCPAddr(a.Scheme, a.Host)
		if err != nil {
			return nil, err
		}

		listener, err = net.ListenTCP(a.Scheme, laddr)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unsupported listener protocol: %s", a.Scheme)
	}

	return &TrackingListener{Listener: listener, connections: make(map[net.Conn]bool)}, nil
}

func (l *TrackingListener) Accept() (net.Conn, error) {
	l.wg.Add(1)
	conn, err := l.Listener.Accept()
	if err != nil {
		l.wg.Done()
		return nil, err
	}

	c := &trackedConn{
		Conn:     conn,
		listener: l,
	}

	return c, nil
}

func (l *TrackingListener) WaitForChildren() {
	l.wg.Wait()
	logger.Log(D{"fn": "shutdown"})
}

type trackedConn struct {
	net.Conn
	listener *TrackingListener
	once     sync.Once
}

func (c *trackedConn) Close() error {
	c.once.Do(c.listener.wg.Done)

	return c.Conn.Close()
}
