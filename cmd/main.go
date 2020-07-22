package main

import (
	"bytes"
	"github.com/ec-protocol/actor-go/pkg/ec"
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
	connection := ec.NewConnection(i, o)
	handleConnection(c, i, o)
	connection.Init()
	for i := 0; i < 10; i++ {
		fsc := make(chan []byte)
		connection.O <- fsc
		readFile(fsc)
	}

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
		connection := ec.NewConnection(i, o)
		handleConnection(c, i, o)
		connection.Init()

		for i := 0; i < 10; i++ {
			writeFile(<-connection.I)
		}
	}
}

func handleConnection(c net.Conn, i chan []byte, o chan []byte) {
	go handleIn(c, i)
	go handleOut(c, o)
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

func readFile(o chan []byte) {
	buf, _ := ioutil.ReadFile("data/in.mp4")
	buf = escape(buf)
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
	o <- nil
}

func writeFile(c chan []byte) {
	in, _ := ioutil.ReadFile("data/in.mp4")
	buf := make([]byte, 0)
	data := <-c
	buf = append(buf, data...)
	for {
		data := <-c
		if data == nil {
			buf = unescape(buf)
			ioutil.WriteFile("data/out.mp4", buf, 0644)
			if bytes.Compare(buf, in) == 0 {

				println("Done!!!")
			} else {
				println(":(")
			}
			return
		}
		buf = append(buf, data...)
	}
}

func escape(pkg []byte) []byte {
	var escapeByte byte = 7
	r := make([]byte, 0, len(pkg))
	for _, e := range pkg {
		switch e {
		case ec.PkgStart:
			r = append(r, escapeByte)
			r = append(r, 8)
		case ec.PkgEnd:
			r = append(r, escapeByte)
			r = append(r, 9)
		case ec.ControlPkgStart:
			r = append(r, escapeByte)
			r = append(r, 10)
		case ec.ControlPkgEnd:
			r = append(r, escapeByte)
			r = append(r, 11)
		case ec.Ignore:
			r = append(r, escapeByte)
			r = append(r, 12)
		case escapeByte:
			r = append(r, escapeByte)
			r = append(r, escapeByte)
		default:
			r = append(r, e)
		}
	}
	return r
}

func unescape(e []byte) []byte {
	var escapeByte byte = 7
	r := make([]byte, 0, len(e))
	for i := 0; i < len(e); i++ {
		switch e[i] {
		case escapeByte:
			i++
			switch e[i] {
			case 8:
				r = append(r, ec.PkgStart)
			case 9:
				r = append(r, ec.PkgEnd)
			case 10:
				r = append(r, ec.ControlPkgStart)
			case 11:
				r = append(r, ec.ControlPkgEnd)
			case 12:
				r = append(r, ec.Ignore)
			case escapeByte:
				r = append(r, escapeByte)
			}
		default:
			r = append(r, e[i])
		}
	}
	return r
}
