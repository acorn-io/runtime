package k8schannel

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/acorn-io/runtime/pkg/client/term"
	"github.com/gorilla/websocket"
	"github.com/rancher/wrangler/pkg/merr"
)

var Upgrader = &websocket.Upgrader{CheckOrigin: func(req *http.Request) bool {
	return true
}, HandshakeTimeout: 15 * time.Second}

type stream struct {
	cond        sync.Cond
	initialized bool
	buf         []byte
}

type Connection struct {
	needsInit   bool
	conn        *websocket.Conn
	streamsLock sync.Mutex
	streams     map[uint8]*stream
	writeLock   sync.Mutex
	err         error
}

func (c *Connection) ToExecIO(tty bool) *term.ExecIO {
	exit := make(chan term.ExitCode, 1)
	go func() {
		exit <- term.ToExitCode(c.ForStream(3))
	}()

	return &term.ExecIO{
		TTY:      tty,
		Stdin:    c.ForStream(0),
		Stdout:   c.ForStream(1),
		Stderr:   c.ForStream(2),
		ExitCode: exit,
		Resize: func(size term.Size) error {
			data, err := json.Marshal(size)
			if err != nil {
				return err
			}
			_, err = c.Write(4, data)
			return err
		},
	}
}

func NewConnection(conn *websocket.Conn, needsInit bool) *Connection {
	c := &Connection{
		needsInit: needsInit,
		conn:      conn,
		streams:   map[uint8]*stream{},
	}
	go c.read()
	go c.ping()
	return c
}

func (c *Connection) getStream(streamNum uint8) *stream {
	c.streamsLock.Lock()
	defer c.streamsLock.Unlock()

	s, ok := c.streams[streamNum]
	if !ok {
		s = &stream{
			cond: sync.Cond{
				L: &sync.Mutex{},
			},
			initialized: !c.needsInit,
		}
		c.streams[streamNum] = s
	}

	return s
}

func (c *Connection) pushStreamData(streamNum uint8, data []byte) {
	stream := c.getStream(streamNum)
	stream.cond.L.Lock()
	defer stream.cond.L.Unlock()

	if !stream.initialized {
		stream.initialized = true
		return
	}

	stream.buf = append(stream.buf, data...)
	stream.cond.Broadcast()
}

func (c *Connection) read() {
	for {
		_, data, err := c.conn.ReadMessage()
		if len(data) > 0 {
			c.pushStreamData(data[0], data[1:])
		}
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			err = io.EOF
		}
		if err != nil {
			c.err = err
			// ensure all readers are broadcasted
			c.Close()
			return
		}
	}
}

func (c *Connection) Read(streamNum uint8, b []byte) (n int, err error) {
	stream := c.getStream(streamNum)
	stream.cond.L.Lock()
	defer stream.cond.L.Unlock()

	for {
		if len(stream.buf) > 0 {
			n := copy(b, stream.buf)
			stream.buf = stream.buf[n:]
			if len(stream.buf) == 0 && c.err != nil {
				return n, c.err
			}
			return n, nil
		}
		if c.err != nil {
			return 0, c.err
		}
		stream.cond.Wait()
	}
}

func (c *Connection) ping() {
	for cont := true; cont; {
		time.Sleep(time.Minute)
		cont = func() bool {
			c.writeLock.Lock()
			defer c.writeLock.Unlock()
			return c.conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(2*time.Second)) == nil
		}()
	}
}

func (c *Connection) Write(streamNum uint8, b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	// k8s doesn't seem like frames more that 1k, which seems
	// inefficient, but who knows, I just work here.
	if len(b) > 1024 {
		n, err := c.Write(streamNum, b[:1024])
		if err != nil {
			return n, err
		}
		n2, err := c.Write(streamNum, b[1024:])
		return n + n2, err
	}

	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	m, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	if _, err := m.Write([]byte{streamNum}); err != nil {
		return 0, err
	}
	n, err = m.Write(b)
	if err != nil {
		return 0, err
	}

	return n, m.Close()
}

func (c *Connection) Close() (err error) {
	defer func() {
		c.streamsLock.Lock()
		defer c.streamsLock.Unlock()
		for _, stream := range c.streams {
			stream.cond.Broadcast()
		}
	}()
	return c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))
}

func (c *Connection) ForStream(streamNum uint8) net.Conn {
	return &netConn{
		conn:      c,
		streamNum: streamNum,
	}
}

type netConn struct {
	conn      *Connection
	streamNum uint8
}

func (c *netConn) Read(b []byte) (n int, err error) {
	return c.conn.Read(c.streamNum, b)
}

func (c *netConn) Write(b []byte) (n int, err error) {
	return c.conn.Write(c.streamNum, b)
}

func (c *netConn) Close() error {
	return c.conn.Close()
}

func (c *netConn) LocalAddr() net.Addr {
	return c.conn.conn.LocalAddr()
}

func (c *netConn) RemoteAddr() net.Addr {
	return c.conn.conn.RemoteAddr()
}

func (c *netConn) SetDeadline(t time.Time) error {
	err1 := c.conn.conn.SetReadDeadline(t)
	err2 := c.conn.conn.SetWriteDeadline(t)
	return merr.NewErrors(err1, err2)
}

func (c *netConn) SetReadDeadline(t time.Time) error {
	return c.conn.conn.SetReadDeadline(t)
}

func (c *netConn) SetWriteDeadline(t time.Time) error {
	return c.conn.conn.SetWriteDeadline(t)
}

func AddCloseHandler(conn *websocket.Conn) {
	conn.SetCloseHandler(func(code int, text string) error {
		// control messages can only be 125 characters and that includes 2 bytes of padding
		if len(text) > 123 {
			// 120 is 125 (max size) - 2 (padding) - 2 (formatting characters [])
			text = "[" + text[len(text)-120:] + "]"
		}
		message := websocket.FormatCloseMessage(code, text)
		_ = conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		return nil
	})
}
