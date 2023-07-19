package dockerconfig

import (
	"encoding/base64"
	"encoding/json"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/z"
	corev1 "k8s.io/api/core/v1"
)

type entry struct {
	Auth string `json:"auth,omitempty"`
}

func toEntry(username, password string) entry {
	return entry{
		Auth: base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
	}
}

func FromCredential(cred *apiv1.Credential) (*corev1.Secret, error) {
	secret, err := FromCredentialData(map[string][]byte{
		"serverAddress": []byte(cred.ServerAddress),
		"username":      []byte(cred.Username),
		"password":      []byte(z.Dereference(cred.Password)),
	})
	if err != nil {
		return nil, err
	}
	secret.Name = cred.Name
	secret.Namespace = cred.Namespace
	return secret, nil
}

func FromCredentialData(data map[string][]byte) (*corev1.Secret, error) {
	data, err := ToData(string(data["serverAddress"]),
		string(data["username"]),
		string(data["password"]))
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		Data: data,
	}, nil
}

func ToData(server, username, password string) (map[string][]byte, error) {
	data, err := json.Marshal(map[string]any{
		"auths": map[string]entry{
			server: toEntry(username, password),
		},
	})
	return map[string][]byte{
		corev1.DockerConfigJsonKey: data,
	}, err
}
