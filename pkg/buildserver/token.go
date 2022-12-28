package buildserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"golang.org/x/crypto/nacl/box"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	tokenCache = cache.NewTTLStore(func(obj interface{}) (string, error) {
		return string(obj.([]byte)), nil
	}, 30*time.Second)
)

func GetToken(req *http.Request, uuid string, pubKey, privKey *[32]byte) (*Token, error) {
	token := req.Header.Get("X-Acorn-Build-Token")
	if token == "" {
		token = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	_, ok, err := tokenCache.Get(data)
	if err != nil {
		return nil, err
	} else if ok {
		return nil, fmt.Errorf("duplicate token")
	}
	if err := tokenCache.Add(data); err != nil {
		return nil, err
	}

	message, ok := box.OpenAnonymous(nil, data, pubKey, privKey)
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}

	result := &Token{}
	if err := json.Unmarshal(message, result); err != nil {
		return nil, err
	}

	if uuid != "" && result.BuilderUUID != uuid {
		return nil, fmt.Errorf("invalid builder UID %s!=%s", result.BuilderUUID, uuid)
	}

	if time.Since(result.Time.Time) > (10 * time.Second) {
		return nil, fmt.Errorf("expired token")
	}

	return result, nil
}

func CreateToken(builder *apiv1.Builder, build *apiv1.AcornImageBuild, pushRepo string) (string, error) {
	data, err := json.Marshal(Token{
		BuilderUUID: builder.Status.UUID,
		Time:        metav1.Now(),
		Build:       (v1.AcornImageBuildInstance)(*build),
		PushRepo:    pushRepo,
	})
	if err != nil {
		return "", err
	}

	key, err := ToKey(builder.Status.PublicKey)
	if err != nil {
		return "", err
	}

	data, err = box.SealAnonymous(nil, data, &key, nil)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}
