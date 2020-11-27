package serv

import (
	"bytes"
	"dk/base"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sort"
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

func init() {
	ch = make(chan packet, queueCap)
}

func procPackets(cf Config) {
	for {
		var session uint32
		var data []byte
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
				if len(dest) != 2+net.IPv4len && len(dest) != 2+net.IPv6len {
					base.Log("ChunkOPN: invalid destination")
					return
				}
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
					p = packet{ct: base.ChunkCON, buf: append(data, []byte(err.Error())...)}
				} else {
					p = packet{ct: base.ChunkCON, buf: data, conn: conn}
					go func(s uint32, c net.Conn) {
						defer func() {
							if e := recover(); e != nil {
								msg := make([]byte, 4)
								binary.BigEndian.PutUint32(msg, s)
								msg = append(msg, []byte(e.(error).Error())...)
								ch <- packet{ct: base.ChunkCON, buf: msg}
							}
						}()
						data := make([]byte, base.MTU-2)
						for {
							n, err := c.Read(data)
							assert(err)
							assert(base.Send(master, data[:n]))
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
				port := binary.BigEndian.Uint16(data[1:])
				hosts := portScan(port, cf.LanNets, cf.ScanTTL)
				var msg bytes.Buffer
				msg.WriteByte(1)
				je := json.NewEncoder(&msg)
				if len(hosts) == 0 {
					je.Encode(map[string]interface{}{
						"stat": false,
						"mesg": fmt.Sprintf("no host opens port %d", port),
					})
				} else {
					sort.Strings(hosts)
					je.Encode(map[string]interface{}{
						"stat": true,
						"data": hosts,
					})
				}
				if err := base.Reply(master, session, msg.Bytes()); err != nil {
					base.Log("reply(scan#%d): %v", port, err)
				}
			}
		case base.ChunkCON:
			if p.conn == nil {
				base.Log("session %x aborted (%s)", session, string(data))
				bad := peer[session]
				if bad != nil {
					bad.Close()
				}
				delete(peer, session)
				break
			}
			s := peer[session]
			if s == nil {
				base.Log("unregistered session %x abandoned", session)
				p.conn.Close()
				break
			}
			s.Connect(p.conn)
			if err := s.Send(base.ChunkNIL, nil); err != nil {
				base.Log("backlog[%x]: %v", session, err)
				base.Close(master, session) //向控制端通告该后端连接关闭
				delete(peer, session)
			}
		}
	}
}

func serve(conn net.Conn, cf Config) {
	peer = make(map[uint32]*base.Conn)
	master = conn
	for {
		ct, buf, err := base.Recv(master)
		if err != nil {
			base.Log("recv: %v", err)
			return
		}
		ch <- packet{ct: ct, buf: buf}
	}
}
