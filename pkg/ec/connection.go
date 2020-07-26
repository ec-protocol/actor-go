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

const cryptBlockSize = 1024
const decryptBlockSize = cryptBlockSize + 28

type Connection struct {
	i        chan []byte
	o        chan []byte
	resetI   chan bool
	resetO   chan bool
	cancelI  chan bool
	cancelO  chan bool
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
		resetI:   make(chan bool),
		resetO:   make(chan bool),
		cancelI:  make(chan bool),
		cancelO:  make(chan bool),
		controlI: make(chan chan []byte),
		I:        make(chan chan []byte),
		O:        make(chan chan []byte),
	}
}

func (c *Connection) Init() {

	initDone := make(chan bool)

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

		c.resetI <- true
		c.resetO <- true
		initDone <- true
	}()

	go c.handleIn()
	go c.handleOut()

	<-initDone
}

func (c *Connection) handleIn() {
	crypt := false
	if c.ikey != nil && c.okey != nil {
		crypt = true
	}
	pos := 0
	var cc chan []byte = nil
	var ccpc chan []byte = nil
	leftover := make([]byte, 0)
	for {
		select {
		case <-c.resetI:
			go c.handleIn()
			return
		case <-c.cancelI:
			return
		case e := <-c.i:
			e = append(leftover, e...)
			leftover = leftover[:0]
			if crypt {
				//todo ensure to encrypt blocks of a fixed size
				dsec := make([]byte, 0, len(e))
				for i := 0; i < len(e); i += decryptBlockSize {
					//todo ensure to encrypt blocks of a fixed size
					if len(e) >= i+decryptBlockSize {
						buf, _ := decryptSync(e[i:i+decryptBlockSize], c.ikey)
						dsec = append(dsec, buf...)
					} else {
						leftover = append(leftover, e[i:]...)
						break
					}
				}
				e = dsec
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
		case <-c.resetO:
			go c.handleOut()
			return
		case <-c.cancelO:
			return
		case cc = <-c.O:
		}
		b := true
		for cc != nil {
			select {
			case <-c.resetO:
				go c.handleOut()
				return
			case <-c.cancelO:
				return
			case sec := <-cc:
				checkControlCharacters(sec)
				if sec == nil {
					sec = []byte{PkgEnd}
					cc = nil
				}
				if b {
					sec = append([]byte{PkgStart}, sec...)
					b = false
				}
				if crypt {
					esec := make([]byte, 0, len(sec))
					cryptBuf := make([]byte, 0, cryptBlockSize)
					for i := 0; i < len(sec); i += cryptBlockSize {
						//todo ensure to encrypt blocks of a fixed size
						if len(sec) >= i+cryptBlockSize {
							cryptBuf = append(cryptBuf, sec[i:i+cryptBlockSize]...)
						} else {
							cryptBuf = append(cryptBuf, sec[i:]...)
							ignore := make([]byte, cryptBlockSize-len(cryptBuf))
							for i := range ignore {
								ignore[i] = Ignore
							}
							cryptBuf = append(cryptBuf, ignore...)
						}
						block, _ := encryptSync(cryptBuf, c.okey)
						esec = append(esec, block...)
						cryptBuf = cryptBuf[:0]
					}
					sec = esec
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
