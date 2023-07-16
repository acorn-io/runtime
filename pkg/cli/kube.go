package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewKubectl(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Kube{client: c.ClientFactory}, cobra.Command{
		Use:          "kube [flags]",
		Args:         cobra.MinimumNArgs(1),
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
	client ClientFactory
}

func (s *Kube) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	server, err := c.KubeProxyAddress(ctx)
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

	k := exec.Command(args[0], args[1:]...)
	k.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", f.Name()))
	k.Stdin = os.Stdin
	k.Stdout = os.Stdout
	k.Stderr = os.Stderr
	return k.Run()
}
