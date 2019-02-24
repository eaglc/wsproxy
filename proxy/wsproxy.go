package proxy

import (
	"net"
	"sync"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type wsproxy struct {
	down   websocket.Conn
	up     net.Conn
	ws     sync.WaitGroup
	lock   sync.Mutex
	errsig chan bool

	exitsig bool

	fontBuf chan []byte
	backBuf chan []byte
}

func NewWsProxy(up net.Conn, down websocket.Conn) *wsproxy {
	d := &wsproxy{
		down:    down,
		up:      up,
		errsig:  make(chan bool),
		fontBuf: make(chan []byte, 1024),
		backBuf: make(chan []byte, 1024),
		exitsig: true,
	}

	return d
}

func (d *wsproxy) Process() {
	d.exitsig = false
	func() {
		defer d.up.Close()
		defer d.down.Close()

		go d.fontRecv()
		go d.fontWrite()

		go d.backRecv()
		go d.backWrite()

		<-d.errsig
	}()
	d.ws.Wait()
}

func (d *wsproxy) fontRecv() {
	d.ws.Add(1)

	defer close(d.fontBuf)
	defer d.ws.Done()

	for {
		_, msg, err := d.down.ReadMessage()
		if err != nil {
			d.lock.Lock()
			defer d.lock.Unlock()
			if !d.exitsig {
				d.exitsig = true
				d.errsig <- true
			}
			glog.Warningf("fontRecv error: %v", err)
			return
		}
		glog.Infof("recv from client: %v", msg)
		d.fontBuf <- msg
	}
}

func (d *wsproxy) fontWrite() {
	d.ws.Add(1)

	defer d.ws.Done()

	for {
		b, ok := <-d.backBuf
		if !ok {
			return
		}
		err := d.down.WriteMessage(1, b)
		if err != nil {
			d.lock.Lock()
			defer d.lock.Unlock()
			if !d.exitsig {
				d.exitsig = true
				d.errsig <- true
			}
			glog.Warningf("fontWrite error: %v", err)
			return
		}
	}
}

func (d *wsproxy) backRecv() {
	d.ws.Add(1)

	defer close(d.backBuf)
	defer d.ws.Done()

	buf := make([]byte, 1024)
	for {
		n, err := d.up.Read(buf)
		if n == 0 || err != nil {
			d.lock.Lock()
			defer d.lock.Unlock()
			if !d.exitsig {
				d.exitsig = true
				d.errsig <- true
			}
			glog.Warningf("backRecv error: %v", err)
			return
		}

		b := make([]byte, n)
		copy(b, buf[:n])
		d.backBuf <- b
	}
}

func (d *wsproxy) backWrite() {
	d.ws.Add(1)

	defer d.ws.Done()

	for {
		b, ok := <-d.fontBuf
		if !ok {
			return
		}

		glog.Infof("write to server: %v", b)
		_, err := d.up.Write(b)

		if err != nil {
			d.lock.Lock()
			defer d.lock.Unlock()
			if !d.exitsig {
				d.exitsig = true
				d.errsig <- true
			}
			glog.Warningf("backWrite error: %v", err)
			return
		}
	}
}
