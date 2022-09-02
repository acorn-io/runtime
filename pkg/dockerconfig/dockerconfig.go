package dockerconfig

import (
	"encoding/base64"
	"encoding/json"

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
