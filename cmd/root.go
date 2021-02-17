package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"time"

	"github.com/spf13/cobra"

	"github.com/ec-protocol/actor-go/handler"
	"github.com/ec-protocol/actor-go/pkg/ec"
)

var unsafe bool
var connect string
var listen string
var in string
var out string

var rootCmd = &cobra.Command{
	Use:   "actor-go",
	Short: "run actor-go",
	Long:  `run actor-go actor-go is a implementation of the ec protocol written in go`,
	Run:   run,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())

	rootCmd.Flags().BoolVarP(&unsafe, "unsafe", "u", false, "disables encryption")
	rootCmd.Flags().StringVarP(&connect, "connect", "c", "", "address to connect to")
	rootCmd.Flags().StringVarP(&listen, "listen", "l", "", "address to listen on")
	rootCmd.Flags().StringVarP(&in, "in", "i", "", "input file path")
	rootCmd.Flags().StringVarP(&out, "out", "o", "", "out file path")
}

func run(*cobra.Command, []string) {
	isClient := true
	tcpConn, err := net.Dial("tcp", connect)
	if err != nil {
		isClient = false
		tcpAddr, _ := net.ResolveTCPAddr("tcp", listen)
		tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
		tcpConn, _ = tcpListener.Accept()
	}

	conn := openConnection(tcpConn)

	if isClient {
		receiveData(conn)
	} else {
		sendData(conn)
	}
}

func openConnection(c net.Conn) ec.Connection {
	i := make(chan []byte)
	o := make(chan []byte)
	connection := ec.NewConnection(i, o)
	go handler.HandleIn(c, i)
	go handler.HandleOut(c, o)
	connection.Init(!unsafe)
	return connection
}

func receiveData(c ec.Connection) {
	for i := 0; i < 1; i++ {
		writeFile(<-c.In)
	}
}

func writeFile(c chan []byte) {
	in, _ := ioutil.ReadFile(in)
	buf := make([]byte, 0)
	data := <-c
	buf = append(buf, data...)
	for data := range c {
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
		c.Out <- fsc
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
