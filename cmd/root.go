package cmd

import (
	"bytes"
	"fmt"
	"github.com/ec-protocol/actor-go/pkg/ec"
	"github.com/spf13/cobra"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"time"
)

var encrypt bool
var connect string
var listen string
var in string
var out string

var rootCmd = &cobra.Command{
	Use:   "actor-go",
	Short: "run actor-go",
	Long: `run actor-go
actor-go is a implementation of the ec protocol written in go`,
	Run: func(cmd *cobra.Command, args []string) {
		encrypt, _ = cmd.Flags().GetBool("unsafe")
		encrypt = !encrypt
		connect, _ = cmd.Flags().GetString("connect")
		listen, _ = cmd.Flags().GetString("listen")
		in, _ = cmd.Flags().GetString("in")
		out, _ = cmd.Flags().GetString("out")
		run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("unsafe", "u", false, "disables encryption")
	rootCmd.Flags().StringP("connect", "c", "", "address to connect to")
	rootCmd.Flags().StringP("listen", "l", "", "address to listen on")
	rootCmd.Flags().StringP("in", "i", "", "input file path")
	rootCmd.Flags().StringP("out", "o", "", "out file path")
}

func run() {
	isClient := true
	tcpConn, err := net.Dial("tcp", connect)
	if err != nil {
		isClient = false
		tcpAddr, _ := net.ResolveTCPAddr("tcp", listen)
		tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
		tcpConn, _ = tcpListener.Accept()
	}

	conn := createConnection(tcpConn)

	if isClient {
		receiveData(conn)
	} else {
		sendData(conn)
	}
}

func createConnection(c net.Conn) ec.Connection {
	i := make(chan []byte)
	o := make(chan []byte)
	connection := ec.NewConnection(i, o)
	go handleIn(c, i)
	go handleOut(c, o)
	connection.Init(encrypt)
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
		for i := len(buf); i < pkgSize; i = len(buf) {
			if i == 0 {
				b := <-o
				buf = append(buf, b...)
			}
			select {
			case b := <-o:
				buf = append(buf, b...)
			case <-time.After(100 * time.Microsecond):
				break bufContext
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
	for i := 0; i < 1; i++ {
		writeFile(<-c.I)
	}
}

func writeFile(c chan []byte) {
	in, _ := ioutil.ReadFile(in)
	buf := make([]byte, 0)
	data := <-c
	buf = append(buf, data...)
	for {
		data := <-c
		if data == nil {
			buf = unescape(buf)
			ioutil.WriteFile(out, buf, 0644)
			if bytes.Compare(buf, in) == 0 {
				fmt.Println("complete")
			} else {
				fmt.Println("failed")
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
	for i := 0; i < 1; i++ {
		fsc := make(chan []byte)
		c.O <- fsc
		readFile(fsc)
	}
}

func readFile(c chan []byte) {
	buf, _ := ioutil.ReadFile(in)
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
