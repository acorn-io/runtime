package buildclient

import (
	"context"
	"encoding/json"
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
	//         RegistryServerAddress - Server requesting a registry credential, or Client responding

	FileSessionID         string       `json:"fileSessionID,omitempty"`
	StatusSessionID       string       `json:"statusSessionID,omitempty"`
	AppImage              *v1.AppImage `json:"appImage,omitempty"`
	Error                 string       `json:"error,omitempty"`
	Acornfile             string       `json:"acornfile,omitempty"`
	RegistryServerAddress string       `json:"registryServerAddress,omitempty"`

	// The below fields are additional metadata for each one of the above messages types

	FileSessionClose bool                `json:"fileSessionClose,omitempty"`
	RegistryAuth     *apiv1.RegistryAuth `json:"registryAuth,omitempty"`
	SyncOptions      *SyncOptions        `json:"syncOptions,omitempty"`
	Packet           *types.Packet       `json:"packet,omitempty"`
	Status           *client.SolveStatus `json:"status,omitempty"`
}

func (m *Message) String() string {
	data, _ := json.Marshal(m)
	return string(data)
}

type SyncOptions struct {
	Context            string
	Dockerfile         string
	DockerfileContents string

	OverrideExcludes   []string
	IncludePatterns    []string
	ExcludePatterns    []string
	FollowPaths        []string
	DirName            []string
	ExporterMetaPrefix []string
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
		logrus.Tracef("Read build message %s", msg)
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
	logrus.Tracef("Send build message %s", msg)
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.conn.WriteJSON(msg)
}
