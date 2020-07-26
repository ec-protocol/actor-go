package main

import (
	"github.com/ec-protocol/actor-go/cmd"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cmd.Execute()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}
