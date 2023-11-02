package buildclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/acorn-io/aml"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/streams"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tonistiigi/fsutil/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

func wsURL(url string) string {
	if strings.HasPrefix(url, "http") {
		return strings.Replace(url, "http", "ws", 1)
	}
	return url
}

type CredentialLookup func(ctx context.Context, serverAddress string) (*apiv1.RegistryAuth, bool, error)

type WebSocketDialer func(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)

func Stream(ctx context.Context, cwd string, streams *streams.Output, dialer WebSocketDialer,
	creds CredentialLookup, build *apiv1.AcornImageBuild) (*v1.AppImage, error) {
	conn, response, err := dialer(ctx, wsURL(build.Status.BuildURL), map[string][]string{
		"X-Acorn-Build-Token": {build.Status.Token},
	})
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response == nil {
			return nil, err
		}

		// If there was a body and an error occurred, read the body and write it
		// into the error message.
		body, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return nil, fmt.Errorf("%w: status %v, error occurred while reading builder response body: %v", err, response.StatusCode, readErr)
		}
		return nil, fmt.Errorf("%w: %v, %v", err, response.StatusCode, string(body))
	}

	var (
		messages = NewWebsocketMessages(conn)
		syncers  = map[string]*fileSyncClient{}
	)
	defer func() {
		for _, s := range syncers {
			s.Close()
		}
	}()
	defer messages.Close()

	msgs, cancel := messages.Recv()
	defer cancel()

	var (
		progress = newClientProgress(ctx, streams)
		credHits = sets.New[string]()
	)
	defer func() {
		// Close the progress bar first to ensure it doesn't clobber credential hits in stdout.
		progress.Close()

		// If we sent the build server any local credentials, print a message.
		if len(credHits) < 1 || streams.Out == nil {
			return
		}
		if _, err := fmt.Fprintln(streams.Out, "Used local credentials for:", strings.Join(sets.List(credHits), ", ")); err != nil {
			logrus.WithError(err).Error("failed to print credential hits")
		}
	}()

	// Handle messages synchronous since new subscribers are started,
	// and we don't want to miss a message.
	messages.OnMessage(func(msg *Message) error {
		if msg.FileSessionID == "" {
			return nil
		}
		if _, ok := syncers[msg.FileSessionID]; ok {
			return nil
		}
		s, err := newFileSyncClient(ctx, cwd, msg.FileSessionID, messages, msg.SyncOptions)
		if err != nil {
			return err
		}
		syncers[msg.FileSessionID] = s
		return nil
	})

	messages.Start(ctx)

	for msg := range msgs {
		if msg.Status != nil {
			progress.Display(msg)
		} else if msg.AppImage != nil {
			return msg.AppImage, nil
		} else if msg.RegistryServerAddress != "" {
			cm := lookupCred(ctx, creds, msg.RegistryServerAddress)
			if cm != nil && cm.RegistryAuth != nil {
				// Mark the credential as used
				credHits.Insert(msg.RegistryServerAddress)
			}
			if err := messages.Send(cm); err != nil {
				return nil, err
			}
		} else if msg.Acornfile != "" {
			data, err := aml.ReadFile(filepath.Join(cwd, msg.Acornfile))
			if err != nil {
				return nil, err
			}
			err = messages.Send(&Message{
				Acornfile: msg.Acornfile,
				Packet: &types.Packet{
					Data: data,
				},
			})
			if err != nil {
				return nil, err
			}
		} else if msg.ReadFile != "" {
			data, err := os.ReadFile(filepath.Join(cwd, msg.ReadFile))
			if err != nil {
				return nil, err
			}
			err = messages.Send(&Message{
				ReadFile: msg.ReadFile,
				Packet: &types.Packet{
					Data: data,
				},
			})
			if err != nil {
				return nil, err
			}
		} else if msg.Error != "" {
			return nil, errors.New(msg.Error)
		}
	}

	return nil, fmt.Errorf("build failed")
}

func lookupCred(ctx context.Context, creds CredentialLookup, serverAddress string) (result *Message) {
	result = &Message{
		RegistryServerAddress: serverAddress,
	}

	if creds == nil {
		return
	}

	cred, found, err := creds(ctx, serverAddress)
	if err != nil {
		logrus.Errorf("failed to lookup credential for server address %s: %v", serverAddress, err)
		return
	} else if !found {
		return
	}

	result.RegistryAuth = &apiv1.RegistryAuth{
		Username: cred.Username,
		Password: cred.Password,
	}
	return
}

func PingBuilder(ctx context.Context, baseURL string) bool {
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/ping", nil)
		if err != nil {
			logrus.Debugf("failed to build request for builder ping to %s: %v", baseURL+"/ping", err)
			return false
		}

		subCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		resp, err := http.DefaultClient.Do(req.WithContext(subCtx))
		cancel()
		if err != nil {
			logrus.Debugf("builder ping failed: %v", err)
		} else {
			_ = resp.Body.Close()
			logrus.Debugf("builder status code: %d", resp.StatusCode)
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(time.Second)
	}
	return false
}
