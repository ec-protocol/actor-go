package handler

import (
	"net"
	"time"
)

func HandleOut(c net.Conn, o chan []byte) {
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
