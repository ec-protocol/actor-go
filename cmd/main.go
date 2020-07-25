package main

import (
	"bytes"
	"fmt"
	"github.com/ec-protocol/actor-go/pkg/ec"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	connectTo = "localhost:6563"
	listenOn  = ":6563"
	inFile    = "data/in.mp4"
	outFile   = "data/out.mp4"
)

func main() {
	isClient := true
	tcpConn, err := net.Dial("tcp", connectTo)
	if err != nil {
		isClient = false
		tcpAddr, _ := net.ResolveTCPAddr("tcp", listenOn)
		tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
		tcpConn, _ = tcpListener.Accept()
	}

	conn := createConnection(tcpConn)

	if isClient {
		receiveData(conn)
	} else {
		sendData(conn)
	}

	cc := make(chan os.Signal)
	signal.Notify(cc, os.Interrupt, syscall.SIGTERM)
	<-cc
}

func createConnection(c net.Conn) ec.Connection {
	i := make(chan []byte)
	o := make(chan []byte)
	connection := ec.NewConnection(i, o)
	go handleIn(c, i)
	go handleOut(c, o)
	connection.Init()
	return connection
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

func receiveData(c ec.Connection) {
	for i := 0; i < 10; i++ {
		writeFile(<-c.I)
	}
}

func writeFile(c chan []byte) {
	in, _ := ioutil.ReadFile(inFile)
	buf := make([]byte, 0)
	data := <-c
	buf = append(buf, data...)
	for {
		data := <-c
		if data == nil {
			buf = unescape(buf)
			ioutil.WriteFile(outFile, buf, 0644)
			if bytes.Compare(buf, in) == 0 {
				fmt.Print("complete")
			} else {
				fmt.Print("failed")
			}
			return
		}
		buf = append(buf, data...)
	}
}

func unescape(a []byte) []byte {
	var escapeByte byte = 7
	r := make([]byte, 0, len(a))
	for i := 0; i < len(a); i++ {
		switch a[i] {
		case escapeByte:
			i++
			switch a[i] {
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
			r = append(r, a[i])
		}
	}
	return r
}

func sendData(c ec.Connection) {
	for i := 0; i < 10; i++ {
		fsc := make(chan []byte)
		c.O <- fsc
		readFile(fsc)
	}
}

func readFile(c chan []byte) {
	buf, _ := ioutil.ReadFile(inFile)
	buf = escape(buf)
	for len(buf) > 0 {
		l := rand.Intn(100000) + 1
		if l > len(buf) {
			l = len(buf)
		}
		pkg := make([]byte, l)
		copy(pkg, buf[:l])
		c <- pkg
		buf = buf[l:]
	}
	c <- nil
}

func escape(a []byte) []byte {
	var escapeByte byte = 7
	r := make([]byte, 0, len(a))
	for _, e := range a {
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
