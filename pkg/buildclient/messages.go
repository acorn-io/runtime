package buildclient

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/acorn-io/mink/pkg/channel"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/gorilla/websocket"
	"github.com/moby/buildkit/client"
	"github.com/sirupsen/logrus"
	"github.com/tonistiigi/fsutil/types"
)

type Messages interface {
	Recv() (msgs <-chan *Message, cancel func())
	Send(msg *Message) error
	Close()
}

type Message struct {
	// Only one of the following four fields must be set to indicate the message type
	// Fields: FileSessionID - File transfer message
	//         StatusSessionID - Status message
	//         AppImage - Build done, result
	//         Error - Build failed, error
	//         Acornfile - Request/Response for Acornfile lookup
	//         ReadFile - Request/Response for file lookup
	//         RegistryServerAddress - Server requesting a registry credential, or Client responding

	FileSessionID         string       `json:"fileSessionID,omitempty"`
	StatusSessionID       string       `json:"statusSessionID,omitempty"`
	AppImage              *v1.AppImage `json:"appImage,omitempty"`
	Error                 string       `json:"error,omitempty"`
	Acornfile             string       `json:"acornfile,omitempty"`
	ReadFile              string       `json:"readFile,omitempty"`
	RegistryServerAddress string       `json:"registryServerAddress,omitempty"`

	// The below fields are additional metadata for each one of the above messages types

	FileSessionClose bool                `json:"fileSessionClose,omitempty"`
	RegistryAuth     *apiv1.RegistryAuth `json:"registryAuth,omitempty"`
	SyncOptions      *SyncOptions        `json:"syncOptions,omitempty"`
	Packet           *types.Packet       `json:"packet,omitempty"`
	PacketData       []byte              `json:"packetData,omitempty"`
	Status           *client.SolveStatus `json:"status,omitempty"`
	Compress         bool                `json:"compress,omitempty"`
}

type message Message

func (m *Message) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*message)(m)); err != nil {
		return err
	}
	if len(m.PacketData) == 0 {
		return nil
	}

	packetData, err := decompress(m.PacketData)
	if err != nil {
		return err
	}
	p := &types.Packet{}
	if err := p.Unmarshal(packetData); err != nil {
		return err
	}

	m.Packet = p
	m.PacketData = nil
	return nil
}

func decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	newData, err := io.ReadAll(r)
	if err != nil {
		return data, err
	}

	return newData, nil
}

func compress(data []byte) []byte {
	buf := &bytes.Buffer{}
	w := gzip.NewWriter(buf)
	_, err := w.Write(data)
	if err != nil {
		return data
	}

	if err := w.Close(); err != nil {
		return data
	}
	return buf.Bytes()
}

func (m *Message) MarshalJSON() ([]byte, error) {
	if !m.Compress || m.Packet == nil || m.Packet.Type != types.PACKET_DATA {
		return json.Marshal((*message)(m))
	}

	data, err := m.Packet.Marshal()
	if err != nil {
		return nil, err
	}

	data = compress(data)

	cp := (message)(*m)
	cp.PacketData = data
	if len(cp.PacketData) > 0 {
		pcp := *cp.Packet
		pcp.Data = nil
		cp.Packet = &pcp
	}

	return json.Marshal(cp)
}

func (m *Message) String() string {
	data, _ := json.Marshal(m)
	return string(data)
}

type SyncOptions struct {
	Context            string
	AdditionalContexts map[string]string
	Dockerfile         string
	DockerfileContents string

	OverrideExcludes   []string
	IncludePatterns    []string
	ExcludePatterns    []string
	FollowPaths        []string
	DirName            []string
	ExporterMetaPrefix []string
	Compress           bool
}

type WebsocketMessages struct {
	lock        sync.Mutex
	conn        *websocket.Conn
	messages    chan *Message
	handler     func(*Message) error
	ctx         context.Context
	cancel      func()
	broadcaster *channel.Broadcaster[*Message]
}

func NewWebsocketMessages(conn *websocket.Conn) *WebsocketMessages {
	m := &WebsocketMessages{
		conn:     conn,
		messages: make(chan *Message, 10),
	}
	m.broadcaster = channel.NewBroadcaster(m.messages)
	return m
}

// OnMessage is a synchronous handler that will block the input of messages until the
// handler finishes.
func (m *WebsocketMessages) OnMessage(handler func(message *Message) error) {
	if m.handler != nil {
		panic("only one handler is currently supported")
	}
	m.handler = handler
}

func (m *WebsocketMessages) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.ctx = ctx

	go m.broadcaster.Start(ctx)
	go func() {
		err := m.run(ctx)
		if err != nil {
			logrus.Debugf("run loop error: %v", err)
		}
	}()
}

func (m *WebsocketMessages) run(ctx context.Context) error {
	defer m.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg := &Message{}
		if err := m.conn.ReadJSON(msg); err != nil {
			return err
		}
		logrus.Tracef("Read build message %s", redact(msg))
		if m.handler != nil {
			if err := m.handler(msg); err != nil {
				return err
			}
		}
		m.messages <- msg
	}
}

func (m *WebsocketMessages) Close() {
	if m.cancel != nil {
		m.cancel()
	}
	go func() {
		if m.ctx != nil {
			<-m.ctx.Done()
		}
		// Shutdown here, don't close as shutdown will ensure all subscribers still get their messages
		m.broadcaster.Shutdown()
	}()
	m.conn.Close()
}

func (m *WebsocketMessages) Recv() (<-chan *Message, func()) {
	sub := m.broadcaster.Subscribe()
	return sub.C, sub.Close
}

func (m *WebsocketMessages) Send(msg *Message) error {
	logrus.Tracef("Send build message %s", redact(msg))
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.conn.WriteJSON(msg)
}

// redact returns a Message with all sensitive information redacted.
// Use this to prep a Message for logging.
func redact(msg *Message) *Message {
	if msg == nil {
		return nil
	}

	redacted := *msg
	if redacted.RegistryAuth != nil {
		redacted.RegistryAuth = &apiv1.RegistryAuth{
			Username: "REDACTED",
			Password: "REDACTED",
		}
	}

	return &redacted
}
