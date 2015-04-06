package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const (
	contentMediaType = "application/vnd.git-lfs"
	metaMediaType    = contentMediaType + "+json"
	version          = "0.1.0"
)

var (
	logger = NewKVLogger(os.Stdout)
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-v" {
		fmt.Println(version)
		os.Exit(0)
	}

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

	logger.Log(kv{"fn": "main", "msg": "listening", "pid": os.Getpid(), "addr": Config.Listen, "version": version})

	app := NewApp(contentStore, metaStore)
	app.Serve(tl)
	tl.WaitForChildren()
}
