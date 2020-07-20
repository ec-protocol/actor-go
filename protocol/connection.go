package protocol

import "errors"

const (
	PkgStart = iota + 1
	PkgEnd
	ControlPkgStart
	ControlPkgEnd
	Ignore
)

type Connection struct {
	i chan []byte
	o chan []byte
	c chan bool
	I chan chan []byte
	O chan chan []byte
}

func NewConnection(i chan []byte, o chan []byte) Connection {
	return Connection{
		i: i,
		o: o,
		c: make(chan bool),
		I: make(chan chan []byte),
		O: make(chan chan []byte),
	}
}

func (c Connection) Init() {
	go c.handleIn()
	go c.handleOut()
}

func (c Connection) handleIn() {
	pos := 0
	var cc chan []byte = nil
	for {
		select {
		case e := <-c.i:
			sec := make([]byte, 0, len(e))
			controlPkg := make([]byte, 0)
			for _, i := range e {
				switch i {
				case PkgStart:
					if pos != 0 {
						panic(errors.New("illegal state"))
					}
					pos = 1
					cc = make(chan []byte)
					c.I <- cc
				case PkgEnd:
					if pos != 1 {
						panic(errors.New("illegal state"))
					}
					pos = 0
					receiveSec(sec, cc)
					sec = sec[:0]
					cc <- nil
					cc = nil
				case ControlPkgStart:
					if pos != 0 {
						panic(errors.New("illegal state"))
					}
					pos = 2
				case ControlPkgEnd:
					if pos != 2 {
						panic(errors.New("illegal state"))
					}
					pos = 0
				case Ignore:
				default:
					switch pos {
					case 1:
						sec = append(sec, i)
					case 2:
						controlPkg = append(controlPkg, i)
					default:
					}
				}
			}
			receiveSec(sec, cc)
		case <-c.c:
			return
		}
	}
}

func (c Connection) handleOut() {
	for {
		var cc chan []byte = nil
		select {
		case cc = <-c.O:
		case <-c.c:
			return
		}
		c.o <- []byte{PkgStart}
		for cc != nil {
			select {
			case sec := <-cc:
				if sec == nil {
					cc = nil
				}
				checkControlCharacters(sec)
				c.o <- sec
			case <-c.c:
				return
			}
		}
		c.o <- []byte{PkgEnd}
	}
}

func receiveSec(sec []byte, cc chan []byte) {
	l := len(sec)
	if cc != nil && l > 0 {
		s := make([]byte, l)
		copy(s, sec[:l])
		cc <- s
	}
}

func checkControlCharacters(e []byte) {
	for _, i := range e {
		switch i {
		case PkgStart, PkgEnd, ControlPkgStart, ControlPkgEnd, Ignore:
			panic(errors.New("bytes values from 1 until 5 used as control characters and therefor can not be send or must be encode or escape"))
		default:
		}
	}
}
