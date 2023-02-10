package credentials

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	credentials2 "github.com/acorn-io/acorn/pkg/server/registry/credentials"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	credentials3 "github.com/docker/docker-credential-helpers/credentials"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Credential struct {
	ServerAddress string
	Username      string
	Password      string
	LocalStorage  bool
}

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

func IsErrCredentialsNotFound(err error) bool {
	return credentials3.IsErrCredentialsNotFound(err)
}

func normalize(cred Credential) Credential {
	cred.ServerAddress = imagesystem.NormalizeServerAddress(cred.ServerAddress)
	return cred
}

func (s *Store) Get(ctx context.Context, serverAddress string) (*apiv1.RegistryAuth, bool, error) {
	serverAddress = imagesystem.NormalizeServerAddress(serverAddress)
	store, err := s.getStore(serverAddress)
	if err != nil {
		return nil, false, err
	}
	auth, err := store.Get(serverAddress)
	if IsErrCredentialsNotFound(err) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	} else if auth.Password == "" {
		return nil, false, nil
	}
	return &apiv1.RegistryAuth{
		Username: auth.Username,
		Password: auth.Password,
	}, true, nil
}

func (s *Store) Add(ctx context.Context, cred Credential, skipChecks bool) error {
	cred = normalize(cred)
	if cred.LocalStorage {
		if !skipChecks {
			err := credentials2.CredentialValidate(ctx, cred.Username, cred.Password, cred.ServerAddress)
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
			Password:      cred.Password,
			ServerAddress: cred.ServerAddress,
		})
	}

	existing, err := s.c.CredentialGet(ctx, cred.ServerAddress)
	if apierror.IsNotFound(err) {
		_, err = s.c.CredentialCreate(ctx, cred.ServerAddress, cred.Username, cred.Password, skipChecks)
		if err != nil {
			return err
		}

		return nil
	}

	existing.Username = cred.Username
	existing.Password = &cred.Password
	_, err = s.c.CredentialUpdate(ctx, cred.ServerAddress, cred.Username, cred.Password, skipChecks)
	return err
}

func (s *Store) Remove(ctx context.Context, cred Credential) error {
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

func (s *Store) List(ctx context.Context) (result []Credential, err error) {
	creds, err := s.c.CredentialList(ctx)
	for _, cred := range creds {
		result = append(result, Credential{
			ServerAddress: cred.ServerAddress,
			Username:      cred.Username,
		})
	}

	for _, entry := range typed.Sorted(s.cfg.Auths) {
		result = append(result, Credential{
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
			result = append(result, Credential{
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
