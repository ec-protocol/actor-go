package ec

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"errors"
)

const (
	PkgStart byte = iota + 1
	PkgEnd
	ControlPkgStart
	ControlPkgEnd
	Ignore
)

type Connection struct {
	i        chan []byte
	o        chan []byte
	cancel   chan bool
	reset    chan bool
	ikey     []byte
	okey     []byte
	controlI chan chan []byte
	I        chan chan []byte
	O        chan chan []byte
}

func NewConnection(i chan []byte, o chan []byte) Connection {
	return Connection{
		i:        i,
		o:        o,
		cancel:   make(chan bool),
		controlI: make(chan chan []byte),
		I:        make(chan chan []byte),
		O:        make(chan chan []byte),
	}
}

func (c *Connection) Init() {
	pk, _ := genRSAKey(2048)
	spk := x509.MarshalPKCS1PublicKey(&pk.PublicKey)
	spk = escape(spk)
	pkp := make([]byte, 0, len(spk)+2)
	pkp = append(pkp, ControlPkgStart)
	pkp = append(pkp, spk...)
	pkp = append(pkp, ControlPkgEnd)
	c.o <- pkp

	go func() {
		cpc := <-c.controlI
		opkp := make([]byte, 0)
		for {
			data := <-cpc
			if data == nil {
				break
			}
			opkp = append(opkp, data...)
		}
		opkp = unescape(opkp)
		opk, _ := x509.ParsePKCS1PublicKey(opkp)
		c.ikey = genSyncKey(32)
		epk, _ := rsa.EncryptOAEP(sha256.New(), rand.Reader, opk, c.ikey, nil)
		epk = escape(epk)
		epkp := make([]byte, 0, len(epk)+2)
		epkp = append(epkp, ControlPkgStart)
		epkp = append(epkp, epk...)
		epkp = append(epkp, ControlPkgEnd)
		c.o <- epkp

		cpc = <-c.controlI
		opkp = make([]byte, 0)
		for {
			data := <-cpc
			if data == nil {
				break
			}
			opkp = append(opkp, data...)
		}

		c.okey, _ = rsa.DecryptOAEP(sha256.New(), rand.Reader, pk, unescape(opkp), nil)
		c.reset <- true
	}()

	go c.handleIn()
	go c.handleOut()
}

func (c *Connection) handleIn() {
	crypt := false
	if c.ikey != nil && c.okey != nil {
		crypt = true
	}
	pos := 0
	var cc chan []byte = nil
	var ccpc chan []byte = nil
	for {
		select {
		case <-c.reset:
			go c.handleIn()
			return
		case <-c.cancel:
			return
		case e := <-c.i:
			if crypt {
				e, _ = decryptSync(e, c.ikey)
			}
			sec := make([]byte, 0, len(e))
			csec := make([]byte, 0)
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
					ccpc = make(chan []byte)
					c.controlI <- ccpc
				case ControlPkgEnd:
					if pos != 2 {
						panic(errors.New("illegal state"))
					}
					pos = 0
					receiveSec(csec, ccpc)
					csec = csec[:0]
					ccpc <- nil
					ccpc = nil
				case Ignore:
				default:
					switch pos {
					case 1:
						sec = append(sec, i)
					case 2:
						csec = append(csec, i)
					default:
					}
				}
			}
			receiveSec(sec, cc)
			receiveSec(csec, ccpc)
		}
	}
}

func (c *Connection) handleOut() {
	crypt := false
	if c.ikey != nil && c.okey != nil {
		crypt = true
	}
	for {
		var cc chan []byte = nil
		select {
		case <-c.reset:
			go c.handleOut()
			return
		case <-c.cancel:
			return
		case cc = <-c.O:
		}
		b := true
		for cc != nil {
			select {
			case <-c.reset: //todo consider removing reset option at this state to implement ctf challenge
				go c.handleOut()
				return
			case <-c.cancel:
				return
			case sec := <-cc:
				if sec == nil {
					c.o <- []byte{PkgEnd}
					cc = nil
					break
				}
				checkControlCharacters(sec)
				if b {
					c.o <- append([]byte{PkgStart}, sec...)
					b = false
					break
				}
				if crypt {
					if len(sec) < 12 {

					}
					sec, _ = encryptSync(sec, c.okey)
				}
				c.o <- sec
			}
		}
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
