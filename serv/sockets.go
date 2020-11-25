package serv

import (
	"dk/base"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const (
	queueCap = 1024 //包处理队列长度
)

type (
	packet struct {
		ct   base.ChunkType
		buf  []byte
		conn net.Conn
	}
)

var (
	master net.Conn
	peer   map[uint32]*base.Conn //维护所有目标连接，索引为SESSION-ID
	ch     chan packet
)

func procPackets() {
	var (
		session uint32
		data    []byte
	)
	for {
		session = 0
		data = nil
		p := <-ch
		if len(p.buf) >= 4 {
			session = binary.BigEndian.Uint32(p.buf[:4])
			data = p.buf[4:]
		}
		switch p.ct {
		case base.ChunkCLS:
			c := peer[session]
			if c == nil {
				base.Dbg("session %x not found, cannot finish", session)
				break
			}
			c.Close()
			delete(peer, session)
			base.Dbg("backend finished session %x", session)
		case base.ChunkOPN:
			old := peer[session]
			if old != nil {
				old.Close()
				delete(peer, session)
			}
			peer[session] = base.NewConn(nil)
			go func(session uint32, dest []byte) {
				port := binary.BigEndian.Uint16(dest[:2])
				ip := net.IP(dest[2:])
				addr := fmt.Sprintf("%s:%d", ip.String(), port)
				base.Dbg("open session %x => %s", session, addr)
				d := net.Dialer{Timeout: time.Duration(base.TIMEOUT) * time.Second}
				conn, err := d.Dial("tcp", addr)
				data := make([]byte, 4)
				binary.BigEndian.PutUint32(data, session)
				var p packet
				if err != nil {
					p = packet{ct: base.ChunkNUL, buf: append(data, []byte(err.Error())...)}
				} else {
					p = packet{ct: base.ChunkNUL, buf: data, conn: conn}
					go func(s uint32, c net.Conn) {
						data := make([]byte, base.MTU-2)
						for {
							n, err := c.Read(data)
							if err != nil {
								msg := make([]byte, 4)
								binary.BigEndian.PutUint32(msg, s)
								ch <- packet{ct: base.ChunkNUL, buf: append(msg, []byte(err.Error())...)}
								return
							}
							assert(err)
							buf, _ := base.Encode(base.ChunkDAT, data[:n])
							base.Send(master, buf)
						}
					}(session, conn)
				}
				ch <- p
			}(session, data)
		case base.ChunkDAT:
			c := peer[session]
			if c == nil {
				base.Dbg("dispatch[%x]: dropped %d bytes", session, len(data))
				base.Close(master, session) //向控制端通告该后端连接关闭
				break
			}
			if err := c.Send(base.ChunkDAT, data); err != nil {
				base.Log("dispatch[%x]: %v", session, err)
				if err != base.ErrInvalidChunk {
					base.Close(master, session) //向控制端通告该后端连接关闭
					delete(peer, session)
				}
			}
		case base.ChunkCMD:
			switch data[0] {
			case 0:
				base.Dbg("received ping from gateway")
				if err := base.Ping(master); err != nil {
					base.Log("pong: %v", err)
				}
			case 1:
				base.Log("TODO: handle port scan request")
			}
		case base.ChunkNUL:
			if p.conn == nil {
				base.Log("session %x aborted (%s)", session, string(data))
				bad := peer[session]
				if bad != nil {
					bad.Close()
				}
				delete(peer, session)
				return
			}
			s := peer[session]
			if s == nil {
				base.Log("unregistered session %x abandoned", session)
				p.conn.Close()
				return
			}
			s.Connect(p.conn)
			if err := s.Send(base.ChunkNUL, nil); err != nil {
				base.Log("backlog[%x]: %v", session, err)
				base.Close(master, session) //向控制端通告该后端连接关闭
				delete(peer, session)
			}
		}
	}
}

func serve(conn net.Conn) {
	ch = make(chan packet, queueCap)
	peer = make(map[uint32]*base.Conn)
	master = conn
	go procPackets()
	for {
		ct, buf, err := base.Recv(master)
		if err != nil {
			base.Log("recv: %v", err)
			return
		}
		ch <- packet{ct: ct, buf: buf}
	}
}
