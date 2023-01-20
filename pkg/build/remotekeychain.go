package build

import (
	"fmt"
	"sync"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/buildclient"
	images2 "github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/imagesystem"
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

func NewRemoteKeyChain(messages buildclient.Messages, next authn.Keychain) *RemoteKeyChain {
	keychain := &RemoteKeyChain{
		messages: messages,
		next:     next,
		cond:     sync.NewCond(&sync.Mutex{}),
	}
	go keychain.run()
	return keychain
}

func (r *RemoteKeyChain) run() {
	msgs, cancel := r.messages.Recv()
	defer cancel()

	for msg := range msgs {
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

	sent := false
	for {
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

		r.cond.Wait()
	}

	cred := r.creds[resource.RegistryStr()]
	if cred == nil {
		return r.next.Resolve(resource)
	}
	return images2.NewSimpleKeychain(resource, *cred, r.next).Resolve(resource)
}
