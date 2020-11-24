package serv

import (
	"dk/base"
	"encoding/binary"
	"net"
	"sync"
)

const (
	//TODO: 是否需要配置这个超时？
	connWait = 60000 //等待后端连接的时长，以毫秒计算
)

type (
	socketRegistry struct {
		self *base.Conn
		peer map[uint32]*base.Conn //维护所有目标连接，索引为SESSION-ID
		sync.Mutex
	}
)

var sr socketRegistry

func finish(session uint32) {
	sr.Lock()
	defer sr.Unlock()
	c := sr.peer[session]
	if c == nil {
		base.Dbg("session %x not found, cannot finish", session)
		return
	}
	c.Close()
	delete(sr.peer, session)
	base.Dbg("backend finished session %x", session)
}

func open(session uint32, dest net.IP) {
	sr.Lock()
	defer sr.Unlock()
	old := sr.peer[session]
	if old != nil {
		old.Close()
		delete(sr.peer, session)
	}
	//TODO...
}

func dispatch(session uint32, data []byte) {
	sr.Lock()
	defer sr.Unlock()
	c := sr.peer[session]
	if c == nil {
		base.Dbg("dispatch[%x]: session not found, dropped %d bytes", session, len(data))
		return
	}
	err := c.Connect(nil, connWait)
	if err != nil {
		base.Log("dispatch[%x]: %v", session, err)
		return
	}
	err = c.Send(0, base.ChunkDAT, data)
	if err != nil {
		base.Log("dispatch[%x]: %v", session, err)
	}
}

func serve(conn net.Conn) {
	dc := base.NewConn(conn)
	sr = socketRegistry{self: dc, peer: make(map[uint32]*base.Conn)}
	for {
		ct, buf, err := dc.Recv(0)
		if err != nil {
			base.Log("recv: %v", err)
			return
		}
		switch ct {
		case base.ChunkCLS:
			session := binary.BigEndian.Uint32(buf[:4])
			go finish(session)
		case base.ChunkOPN:
			session := binary.BigEndian.Uint32(buf[:4])
			dest := net.IP(buf[4:])
			go open(session, dest)
		case base.ChunkDAT:
			session := binary.BigEndian.Uint32(buf[:4])
			go dispatch(session, buf[4:])
		case base.ChunkCMD:
			switch buf[0] {
			case 0:
				base.Dbg("received ping from gateway")
				dc.Send(0, base.ChunkCMD, []byte{0})
			case 1:
				base.Log("TODO: handle port scan request")
			}
		}
	}
}
