package main

import (
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

const (
	contentMediaType = "application/vnd.git-media"
	metaMediaType    = contentMediaType + "+json"
)

var (
	logger = NewKVLogger(os.Stdout)
)

func main() {
	var listener net.Listener

	a, err := url.Parse(Config.Address)
	if err != nil {
		log.Fatalf("Could not parse listen address: %s, %s", Config.Address, err)
	}

	switch a.Scheme {
	case "fd":
		fd, err := strconv.Atoi(a.Host)
		if err != nil {
			logger.Fatalf("invalid file descriptor: %s", a.Host)
		}

		f := os.NewFile(uintptr(fd), "harbour")
		listener, err = net.FileListener(f)
		if err != nil {
			logger.Fatalf("Can't listen on fd address: %s, %s", Config.Address, err)
		}
	case "tcp", "tcp4", "tcp6":
		laddr, err := net.ResolveTCPAddr(a.Scheme, a.Host)
		if err != nil {
			logger.Fatalf("Could not resolve listen address: %s, %s", Config.Address, err)
		}

		listener, err = net.ListenTCP(a.Scheme, laddr)
		if err != nil {
			logger.Fatalf("Can't listen on address %s, %s", Config.Address, err)
		}
	default:
		logger.Fatalf("Unsupported listener protocol: %s", a.Scheme)
	}

	tl := NewTrackingListener(listener)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func(c chan os.Signal, listener net.Listener) {
		for {
			sig := <-c
			switch sig {
			case syscall.SIGHUP: // Graceful shutdown
				tl.Close()
			}
		}
	}(c, tl)

	logger.Log(D{"fn": "main", "msg": "listening", "pid": os.Getpid(), "scheme": Config.Scheme, "host": Config.Host})

	app := NewApp(&S3Redirector{}, &MetaStore{})
	app.Serve(tl)
	tl.WaitForChildren()
}
