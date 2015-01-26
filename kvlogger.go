package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	pid      int
	hostname string
)

func init() {
	pid = os.Getpid()
	h, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	} else {
		hostname = h
	}
}

type D map[string]interface{}

type KVLogger struct {
	w  io.Writer
	mu sync.Mutex
}

func NewKVLogger(out io.Writer) *KVLogger {
	return &KVLogger{w: out}
}

func (l *KVLogger) Log(data D) {
	var file string
	var line int
	var ok bool

	_, file, line, ok = runtime.Caller(2)
	if ok {
		file = path.Base(file)
	} else {
		file = "???"
		line = 0
	}

	out := fmt.Sprintf("%s %s harbour[%d] [%s:%d]: ", time.Now().UTC().Format(time.RFC3339), hostname, pid, file, line)
	var vals []string

	for k, v := range data {
		vals = append(vals, fmt.Sprintf("%s=%v", k, v))
	}
	out += strings.Join(vals, " ")

	l.mu.Lock()
	fmt.Fprint(l.w, out+"\n")
	l.mu.Unlock()
}

func (l *KVLogger) Fatal(data D) {
	l.Log(data)
	os.Exit(1)
}

func (l *KVLogger) Fatalf(format string, v ...interface{}) {
	l.Fatal(D{"err": fmt.Sprintf(format, v...)})
}
