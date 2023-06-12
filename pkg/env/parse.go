package env

import (
	"context"
	"fmt"
	"os"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/baaah/pkg/randomtoken"
)

func ParseEnvForCLIAndCreateSecret(ctx context.Context, client client.Client, s ...string) (result []v1.NameValue, _ error) {
	random, err := randomtoken.Generate()
	if err != nil {
		return nil, err
	}
	secretName := "shell-env-" + random[:8]
	secretValues := map[string][]byte{}
	for _, s := range s {
		k, v, ok := strings.Cut(s, "=")
		if ok {
			result = append(result, v1.NameValue{
				Name:  k,
				Value: v,
			})
		} else {
			v = os.Getenv(k)
			if v == "" {
				result = append(result, v1.NameValue{
					Name:  k,
					Value: v,
				})
			} else {
				result = append(result, v1.NameValue{
					Name:  k,
					Value: fmt.Sprintf("@{secrets.external:%s.%s}", secretName, k),
				})
				secretValues[k] = []byte(v)
			}
		}
	}

	if len(secretValues) > 0 {
		_, err := client.SecretCreate(ctx, secretName, "", secretValues)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
