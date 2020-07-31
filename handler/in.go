package handler

import "net"

func HandleIn(c net.Conn, i chan []byte) {
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
