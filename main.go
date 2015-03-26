package main

import (
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
		logger.Fatal(kv{"fn": "main", "err": "Could not create listener: " + err.Error()})
	}

	metaStore, err := NewMetaStore(Config.MetaDB)
	if err != nil {
		logger.Fatal(kv{"fn": "main", "err": "Could not open the meta store: " + err.Error()})
	}

	contentStore, err := NewContentStore(Config.ContentPath)
	if err != nil {
		logger.Fatal(kv{"fn": "main", "err": "Could not open the content store: " + err.Error()})
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

	logger.Log(kv{"fn": "main", "msg": "listening", "pid": os.Getpid(), "addr": Config.Listen})

	app := NewApp(contentStore, metaStore)
	app.Serve(tl)
	tl.WaitForChildren()
}
