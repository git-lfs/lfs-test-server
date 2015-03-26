package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const (
	contentMediaType = "application/vnd.git-lfs"
	metaMediaType    = contentMediaType + "+json"
)

var (
	logger = NewKVLogger(os.Stdout)
)

func main() {
	tl, err := NewTrackingListener(Config.Listen)
	if err != nil {
		log.Fatalf("Could not create listener: %s", err)
	}

	metaStore, err := NewMetaStore("lfs.db")
	if err != nil {
		log.Fatalf("Could not open the meta store: %s", err)
	}

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

	logger.Log(D{"fn": "main", "msg": "listening", "pid": os.Getpid(), "addr": Config.Listen})

	app := NewApp(&S3Redirector{}, metaStore)
	app.Serve(tl)
	tl.WaitForChildren()
}
