package main

import (
	"bufio"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	c, err := net.Dial("tcp", "localhost:6563")
	if err != nil {
		startServer()
	}
	i := make(chan []byte)
	o := make(chan []byte)
	handleStdIO(c, i, o)
	go func() {
		for i := 0; i < 10000000000; i++ {
			o <- make([]byte, rand.Intn(1000000)+1)
		}
	}()
	waitOnTermination()
}

func waitOnTermination() {
	cc := make(chan os.Signal)
	signal.Notify(cc, os.Interrupt, syscall.SIGTERM)
	<-cc
}

func startServer() {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", ":6563")
	listener, _ := net.ListenTCP("tcp", tcpAddr)
	for {
		c, err := listener.Accept()
		if err != nil {
			continue
		}
		i := make(chan []byte)
		o := make(chan []byte)
		go handleStdIO(c, i, o)
	}
}

func handleStdIO(c net.Conn, i chan []byte, o chan []byte) {
	go readFromKeyboard(o)
	go writeToStdOut(i)
	go handleIn(c, i)
	go handleOut(c, o)
}

func readFromKeyboard(c chan []byte) {
	r := bufio.NewReader(os.Stdin)
	for {
		t, _ := r.ReadString('\n')
		b := []byte(t)
		c <- b
	}
}

func writeToStdOut(c chan []byte) {
	j := 0
	for {
		for i := 0; i < 1073741824; i++ {
			i += len(<-c)
		}
		j++
		println(j)
	}
}

func handleIn(c net.Conn, i chan []byte) {
	defer c.Close()

	buf := make([]byte, 64*1024)
	for {
		n, _ := c.Read(buf)
		if n <= 0 {
			continue
		}
		i <- buf[:n]
	}
}

func handleOut(c net.Conn, o chan []byte) {
	defer c.Close()

	const pkgSize = 64*1024 - 512
	buf := make([]byte, 0, pkgSize)
	for {
	bufContext:
		for i := 0; i < pkgSize; i++ {
			if i != 0 || len(buf) > 0 {
				select {
				case b := <-o:
					buf = append(buf, b...)
				case <-time.After(100 * time.Microsecond):
					break bufContext
				}
			} else {
				b := <-o
				buf = append(buf, b...)
			}
		}
		if len(buf) <= pkgSize {
			c.Write(buf)
			buf = buf[:0]
		} else {
			for len(buf) > pkgSize {
				c.Write(buf[0:pkgSize])
				buf = buf[pkgSize:]
			}
		}
	}
}
