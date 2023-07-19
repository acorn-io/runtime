package build

import (
	"context"
	"fmt"
	"sync"
	"time"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/buildclient"
	images2 "github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sirupsen/logrus"
)

type RemoteKeyChain struct {
	messages buildclient.Messages
	cond     *sync.Cond
	creds    map[string]*apiv1.RegistryAuth
	next     authn.Keychain
}

func NewRemoteKeyChain(ctx context.Context, messages buildclient.Messages, next authn.Keychain) *RemoteKeyChain {
	keychain := &RemoteKeyChain{
		messages: messages,
		next:     next,
		cond:     sync.NewCond(&sync.Mutex{}),
	}
	go keychain.run(ctx)
	return keychain
}

func (r *RemoteKeyChain) run(ctx context.Context) {
	msgs, cancel := r.messages.Recv()
	// Ensure we broadcast to wake up any waiting goroutines
	defer func() {
		cancel()
		go func() {
			for range msgs {
			}
		}()
		r.cond.L.Lock()
		r.cond.Broadcast()
		r.cond.L.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			logrus.Debugf("context cancelled, closing messages channel")
			return
		case msg, ok := <-msgs:
			if !ok {
				logrus.Debugf("channel closed, done...")
				return
			}

			if msg.RegistryServerAddress != "" {
				r.cond.L.Lock()
				if r.creds == nil {
					r.creds = map[string]*apiv1.RegistryAuth{}
				}
				r.creds[msg.RegistryServerAddress] = msg.RegistryAuth
				r.cond.Broadcast()
				r.cond.L.Unlock()
			}
		}
	}
}

func (r *RemoteKeyChain) Resolve(resource authn.Resource) (authenticator authn.Authenticator, err error) {
	r.cond.L.Lock()
	defer r.cond.L.Unlock()

	defer func() {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			if err == nil {
				config, configErr := authenticator.Authorization()
				if configErr == nil {
					logrus.Debugf("cred lookup for %s resolved to username=%s, password length=%d", resource.RegistryStr(), config.Username, len(config.Password))
				} else {
					logrus.Debugf("cred lookup for %s resolved but failed: %v", resource.RegistryStr(), configErr)
				}
			} else {
				logrus.Debugf("cred lookup for %s has failed: %v", resource.RegistryStr(), err)
			}
		}
	}()

	address := imagesystem.NormalizeServerAddress(resource.RegistryStr())
	if resource.RegistryStr() != address {
		resource, err = name.NewRegistry(address)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize registry from %s to %s", resource.RegistryStr(), address)
		}
	}

	var (
		sent    bool
		moveOn  = make(chan struct{})
		timeout = time.AfterFunc(15*time.Second, func() {
			logrus.Debugf("timed out waiting for credentials for %s", resource.RegistryStr())
			r.cond.L.Lock()
			// Close the moveOn channel so the below for loop will break
			close(moveOn)
			r.cond.Broadcast()
			r.cond.L.Unlock()
		})
	)
	defer timeout.Stop()

outer:
	for {
		logrus.Debugf("checking for credentials for %s", resource.RegistryStr())
		if _, ok := r.creds[resource.RegistryStr()]; ok {
			break
		}

		if !sent {
			err := r.messages.Send(&buildclient.Message{
				RegistryServerAddress: resource.RegistryStr(),
			})
			if err != nil {
				return nil, err
			}
			sent = true
		}

		select {
		case <-moveOn:
			break outer
		default:
		}

		// This blocks until we are told to wake up.
		logrus.Debugf("waiting for credentials for %s", resource.RegistryStr())
		r.cond.Wait()
	}

	cred := r.creds[resource.RegistryStr()]
	if cred == nil {
		return r.next.Resolve(resource)
	}
	return images2.NewSimpleKeychain(resource, *cred, r.next).Resolve(resource)
}
