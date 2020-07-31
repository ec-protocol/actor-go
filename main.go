package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ec-protocol/actor-go/cmd"
)

func main() {
	cmd.Execute()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}
