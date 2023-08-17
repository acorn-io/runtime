package manager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/randomtoken"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/strings/slices"
)

func IsManager(cfg *config.CLIConfig, address string) (bool, error) {
	if slices.Contains(cfg.AcornServers, address) {
		return true, nil
	}

	req, err := http.NewRequest(http.MethodGet, toDiscoverURL(address), nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if !strings.Contains(string(data), "TokenRequest") {
		return false, nil
	}

	cfg.AcornServers = append(cfg.AcornServers, address)
	return true, cfg.Save()
}

func Projects(ctx context.Context, address, token string) ([]string, error) {
	memberships := &membershipList{}
	result := sets.NewString()
	err := httpGet(ctx, toProjectMembershipURL(address), token, memberships)
	if err != nil {
		return nil, err
	}
	for _, membership := range memberships.Items {
		if membership.AccountEndpointURL != "" {
			result.Insert(fmt.Sprintf("%s/%s/%s", address, membership.AccountName, membership.ProjectName))
		}
	}
	return result.List(), nil
}

func ProjectURL(ctx context.Context, serverAddress, accountName, token string) (url string, err error) {
	obj := &account{}
	if err := httpGet(ctx, toAccountURL(serverAddress, accountName), token, obj); err != nil {
		return "", err
	}
	if obj.Status.EndpointURL == "" {
		return "", fmt.Errorf("failed to find endpoint URL for account %s, account may still be provisioning", accountName)
	}
	return obj.Status.EndpointURL, nil
}

func Login(ctx context.Context, cfg *config.CLIConfig, password, address string) (user string, pass string, err error) {
	passwordIsSpecified := password != ""
	if !passwordIsSpecified {
		password, err = randomtoken.Generate()
		if err != nil {
			return "", "", err
		}

		url := toLoginURL(address, password)
		_ = browser.OpenURL(url)
		fmt.Printf("\nNavigate your browser to %s and login\n", url)
	}

	tokenRequestURL := toTokenRequestURL(address, password)
	timeout := time.After(5 * time.Minute)
	for {
		select {
		case <-timeout:
			return "", "", fmt.Errorf("timeout getting authentication token")
		default:
		}

		tokenRequest := &tokenRequest{}
		if err := httpGet(ctx, tokenRequestURL, "", tokenRequest); err == nil {
			if tokenRequest.Status.Expired {
				return "", "", fmt.Errorf("token request has expired, please try to login again")
			}
			if tokenRequest.Status.Token != "" {
				httpDelete(ctx, tokenRequestURL, tokenRequest.Status.Token)
				user = tokenRequest.Spec.AccountName
				pass = tokenRequest.Status.Token
				break
			} else {
				logrus.Debugf("tokenRequest.Status.Token is empty")
			}
		} else if passwordIsSpecified && errors.Is(err, ErrTokenNotFound) {
			return "", "", fmt.Errorf("specified token does not exist; please create a token via the web UI or omit the --password flag to request one via your browser")
		} else {
			logrus.Debugf("error getting tokenrequest: %v", err)
		}

		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return "", "", ctx.Err()
		}
	}

	store, err := credentials.NewStore(cfg, nil)
	if err != nil {
		return user, pass, err
	}

	if err = store.Add(ctx, apiv1.Credential{
		ServerAddress: address,
		Username:      user,
		Password:      &pass,
		LocalStorage:  true,
	}, true); err != nil {
		return user, pass, err
	}

	// reload config, could have changed
	if newCfg, err := config.ReadCLIConfig(cfg.AcornConfig, false); err != nil {
		return user, pass, err
	} else {
		*cfg = *newCfg
	}
	return user, pass, nil
}

func DefaultProject(ctx context.Context, address, token string) (string, error) {
	projects, err := Projects(ctx, address, token)
	if err != nil {
		return "", err
	}
	if len(projects) == 0 {
		return "", err
	}
	if slices.Contains(projects, system.DefaultUserNamespace) {
		return system.DefaultUserNamespace, nil
	}
	return projects[0], nil
}
