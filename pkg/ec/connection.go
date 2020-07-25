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
	privateKey, _ := genRSAKey(2048)
	encodedPublicKey := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)
	encodedPublicKey = escape(encodedPublicKey)
	pkg := make([]byte, 0, len(encodedPublicKey)+2)
	pkg = append(pkg, ControlPkgStart)
	pkg = append(pkg, encodedPublicKey...)
	pkg = append(pkg, ControlPkgEnd)
	c.o <- pkg

	go func() {
		cpc := <-c.controlI
		pkg := make([]byte, 0)
		for {
			data := <-cpc
			if data == nil {
				break
			}
			pkg = append(pkg, data...)
		}
		thereEncodedPublicKey := unescape(pkg)
		therePublicKey, _ := x509.ParsePKCS1PublicKey(thereEncodedPublicKey)
		c.ikey = genSyncKey(32)
		encryptedInKey, _ := rsa.EncryptOAEP(sha256.New(), rand.Reader, therePublicKey, c.ikey, nil)
		encryptedInKey = escape(encryptedInKey)
		pkg = make([]byte, 0, len(encryptedInKey)+2)
		pkg = append(pkg, ControlPkgStart)
		pkg = append(pkg, encryptedInKey...)
		pkg = append(pkg, ControlPkgEnd)
		c.o <- pkg

		cpc = <-c.controlI
		pkg = make([]byte, 0)
		for {
			data := <-cpc
			if data == nil {
				break
			}
			pkg = append(pkg, data...)
		}

		encryptedOutKey := unescape(pkg)
		c.okey, _ = rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedOutKey, nil)

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
