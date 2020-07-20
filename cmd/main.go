package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
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
	simulateTraffic(o)
	waitOnTermination()
}

func simulateTraffic(o chan []byte) {
	go func() {
		buf, _ := ioutil.ReadFile("data/in.mp4")
		for len(buf) > 0 {
			l := rand.Intn(100000) + 1
			if l > len(buf) {
				l = len(buf)
			}
			pkg := make([]byte, l)
			copy(pkg, buf[:l])
			o <- pkg
			buf = buf[l:]
		}
	}()
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
	in, _ := ioutil.ReadFile("data/in.mp4")
	buf := make([]byte, 0)
	data := <-c
	buf = append(buf, data...)
	for {
		select {
		case data := <-c:
			buf = append(buf, data...)
		case <-time.After(2000 * time.Millisecond):
			ioutil.WriteFile("data/out.mp4", buf, 0644)
			if bytes.Compare(buf, in) == 0 {
				println("Done!!!")
			} else {
				println(":(")
			}
			return
		}
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
		s := make([]byte, n)
		copy(s, buf[:n])
		i <- s
	}
}

func handleOut(c net.Conn, o chan []byte) {
	defer c.Close()

	const pkgSize = 64*1024 - 60 - 1
	buf := make([]byte, 0, pkgSize)
	for {
	bufContext:
		for i := 0; i < pkgSize; i = len(buf) {
			if i > 0 {
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
