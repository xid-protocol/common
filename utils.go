package common

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/colin-404/logx"
)

var sig = make(chan os.Signal, 1)

func SignalHandler() {

	signal.Notify(sig, syscall.SIGINT, os.Interrupt, syscall.SIGTERM)

	<-sig

	logx.Infof("Received signal: %v", sig)
	os.Exit(0)
}

func GetTimestamp() int64 {
	return time.Now().UnixMilli()
}
