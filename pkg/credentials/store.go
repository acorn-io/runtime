package credentials

import (
	"context"

	"github.com/acorn-io/baaah/pkg/typed"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	credentials2 "github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/credentials"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Store struct {
	cfg *config.CLIConfig
	c   client.Client
}

func NewLocalOnlyStore(cfg *config.CLIConfig) (*Store, error) {
	return NewStore(cfg, nil)
}

func NewStore(cfg *config.CLIConfig, c client.Client) (*Store, error) {
	return &Store{
		cfg: cfg,
		c:   c,
	}, nil
}

func normalize(cred apiv1.Credential) apiv1.Credential {
	cred.ServerAddress = imagesystem.NormalizeServerAddress(cred.ServerAddress)
	return cred
}

func (s *Store) Get(serverAddress string) (*apiv1.RegistryAuth, bool, error) {
	serverAddress = imagesystem.NormalizeServerAddress(serverAddress)
	store, err := s.getStore(serverAddress)
	if err != nil {
		return nil, false, err
	}
	auth, err := store.Get(serverAddress)
	if err != nil {
		return nil, false, err
	} else if auth.Password == "" {
		return nil, false, nil
	}
	return &apiv1.RegistryAuth{
		Username: auth.Username,
		Password: auth.Password,
	}, true, nil
}

func (s *Store) Add(ctx context.Context, cred apiv1.Credential, skipChecks bool) error {
	cred = normalize(cred)
	if cred.LocalStorage {
		if !skipChecks {
			err := credentials2.CredentialValidate(ctx, cred.Username, cred.GetPassword(), cred.ServerAddress)
			if err != nil {
				return err
			}
		}
		s, err := s.getStore(cred.ServerAddress)
		if err != nil {
			return err
		}
		return s.Store(types.AuthConfig{
			Username:      cred.Username,
			Password:      cred.GetPassword(),
			ServerAddress: cred.ServerAddress,
		})
	}

	existing, err := s.c.CredentialGet(ctx, cred.ServerAddress)
	if apierror.IsNotFound(err) {
		_, err = s.c.CredentialCreate(ctx, cred.ServerAddress, cred.Username, cred.GetPassword(), skipChecks)
		if err != nil {
			return err
		}

		return nil
	}

	existing.Username = cred.Username
	existing.Password = cred.Password
	_, err = s.c.CredentialUpdate(ctx, cred.ServerAddress, cred.Username, cred.GetPassword(), skipChecks)
	return err
}

func (s *Store) Remove(ctx context.Context, cred apiv1.Credential) error {
	cred = normalize(cred)
	if !cred.LocalStorage {
		_, err := s.c.CredentialDelete(ctx, cred.ServerAddress)
		if err != nil {
			return err
		}
	}
	store, err := s.getStore(cred.ServerAddress)
	if err != nil {
		return err
	}
	return store.Erase(cred.ServerAddress)
}

func (s *Store) List(ctx context.Context) (result []apiv1.Credential, err error) {
	creds, err := s.c.CredentialList(ctx)
	for _, cred := range creds {
		result = append(result, apiv1.Credential{
			ServerAddress: cred.ServerAddress,
			Username:      cred.Username,
		})
	}

	for _, entry := range typed.Sorted(s.cfg.Auths) {
		result = append(result, apiv1.Credential{
			ServerAddress: entry.Key,
			Username:      entry.Value.Username,
			LocalStorage:  true,
		})
	}

	helpers := sets.NewString()
	if s.cfg.CredentialsStore != "" {
		helpers.Insert(s.cfg.CredentialsStore)
	}
	for storeName := range s.cfg.CredentialHelpers {
		helpers.Insert(storeName)
	}

	for _, helper := range helpers.List() {
		store, err := s.getStoreByHelper(helper)
		if err != nil {
			return nil, err
		}
		auths, err := store.GetAll()
		if err != nil {
			return nil, err
		}
		for _, entry := range typed.Sorted(auths) {
			result = append(result, apiv1.Credential{
				ServerAddress: entry.Key,
				Username:      entry.Value.Username,
				LocalStorage:  true,
			})
		}
	}

	return
}

func (s *Store) getStore(serverAddress string) (credentials.Store, error) {
	helper := s.cfg.CredentialHelpers[serverAddress]
	if helper == "" {
		helper = s.cfg.CredentialsStore
	}

	return s.getStoreByHelper(helper)
}

func (s *Store) getStoreByHelper(helper string) (credentials.Store, error) {
	if helper == "" {
		return credentials.NewFileStore(s.cfg), nil
	}
	return NewHelper(s.cfg, helper)
}
