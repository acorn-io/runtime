package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/spf13/cobra"
)

func NewKubectl(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Kube{client: c.ClientFactory}, cobra.Command{
		Use:          "kube [flags]",
		Hidden:       true,
		SilenceUsage: true,
		Short:        "Run command with KUBECONFIG env set to a generated kubeconfig of the current project",
		Example: `
acorn -j acorn kube k9s
`})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Kube struct {
	client    ClientFactory
	Region    string `usage:"Get access to the cluster supporting that specific region"`
	WriteFile string `usage:"Write kubeconfig to file" short:"w"`
}

func (s *Kube) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	if s.WriteFile != "" {
		data, err := c.KubeConfig(cmd.Context(), &client.KubeProxyAddressOptions{
			Region: s.Region,
		})
		if err != nil {
			return err
		}
		return os.WriteFile(s.WriteFile, data, 0644)
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	server, err := c.KubeProxyAddress(ctx, &client.KubeProxyAddressOptions{
		Region: s.Region,
	})
	if err != nil {
		return err
	}

	f, err := os.CreateTemp("", "acorn-kube")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(f.Name())
	}()

	_, err = f.Write([]byte(fmt.Sprintf(`
apiVersion: v1
clusters:
- cluster:
    server: "%s"
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: default
`, server)))
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	if len(args) == 0 {
		args = []string{os.Getenv("SHELL")}
	}

	k := exec.Command(args[0], args[1:]...)
	k.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", f.Name()))
	k.Stdin = os.Stdin
	k.Stdout = os.Stdout
	k.Stderr = os.Stderr
	return k.Run()
}
