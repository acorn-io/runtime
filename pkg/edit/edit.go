package edit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/acorn-io/aml"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"k8s.io/kubectl/pkg/cmd/util/editor"
)

var (
	envs = []string{
		"ACORN_EDITOR",
		"EDITOR",
	}
)

func Edit(ctx context.Context, c client.Client, name string, secretOnly bool) error {
	var (
		errs []error
	)

	if !secretOnly {
		app, err := c.AppGet(ctx, name)
		if err == nil {
			return editApp(ctx, c, app)
		}
		errs = append(errs, err)
	}

	secret, err := c.SecretReveal(ctx, name)
	if err == nil {
		return editSecret(ctx, c, secret)
	}

	return errors.Join(append(errs, err)...)
}

func stripComments(buf []byte) []byte {
	result := bytes.Buffer{}
	for _, line := range strings.Split(string(buf), "\n") {
		if strings.HasPrefix(line, "//") {
			continue
		}
		result.WriteString(line)
		result.WriteString("\n")
	}
	return result.Bytes()
}

func commentError(err error, buf []byte) []byte {
	var header bytes.Buffer
	for _, line := range strings.Split(strings.ReplaceAll(err.Error(), "\r", ""), "\n") {
		header.WriteString("// ")
		header.WriteString(line)
		header.WriteString("\n")
	}
	return append(header.Bytes(), buf...)
}

func editSecret(ctx context.Context, c client.Client, secret *apiv1.Secret) error {
	data := map[string]string{}
	for k, v := range secret.Data {
		data[k] = string(v)
	}

	spec, err := aml.Marshal(data)
	if err != nil {
		return err
	}

	editor := editor.NewDefaultEditor(envs)
	for {
		buf, file, err := editor.LaunchTempFile("acorn", "secret.acorn", bytes.NewReader(spec))
		if file != "" {
			_ = os.Remove(file)
		}
		if err != nil {
			return err
		}

		if bytes.Equal(buf, spec) {
			return fmt.Errorf("aborted")
		}

		buf = stripComments(buf)

		data := map[string]string{}
		err = aml.Unmarshal(buf, &data)
		if err != nil {
			spec = commentError(err, buf)
			continue
		}

		kclient, err := c.GetClient()
		if err != nil {
			return err
		}

		dataBytes := map[string][]byte{}
		for k, v := range data {
			dataBytes[k] = []byte(v)
		}
		secret.Data = dataBytes
		err = kclient.Update(ctx, secret)
		if err != nil {
			spec = commentError(err, buf)
			continue
		}

		return nil
	}
}

func editApp(ctx context.Context, c client.Client, app *apiv1.App) error {
	app.Spec.ImageGrantedPermissions = nil
	spec, err := aml.Marshal(app.Spec)
	if err != nil {
		return err
	}

	editor := editor.NewDefaultEditor(envs)
	for {
		buf, file, err := editor.LaunchTempFile("acorn", "acorn.acorn", bytes.NewReader(spec))
		if file != "" {
			_ = os.Remove(file)
		}
		if err != nil {
			return err
		}

		if bytes.Equal(buf, spec) {
			return fmt.Errorf("aborted")
		}

		buf = stripComments(buf)

		app.Spec = v1.AppInstanceSpec{}
		err = aml.Unmarshal(buf, &app.Spec)
		if err != nil {
			spec = commentError(err, buf)
			continue
		}

		kclient, err := c.GetClient()
		if err != nil {
			return err
		}

		err = kclient.Update(ctx, app)
		if err != nil {
			spec = commentError(err, buf)
			continue
		}

		return nil
	}
}
