package main

import (
	"net"
	"sync"
)

type TrackingListener struct {
	wg          sync.WaitGroup
	connections map[net.Conn]bool
	cm          sync.Mutex
	net.Listener
}

func NewTrackingListener(l net.Listener) *TrackingListener {
	return &TrackingListener{Listener: l, connections: make(map[net.Conn]bool)}
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
	logger.Print("Shut down")
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
