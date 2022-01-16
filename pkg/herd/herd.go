package herd

import (
	"strings"

	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func sheep() string {
	// Artist:  Bob Allison
	return strings.ReplaceAll(`
           __  _
       .-.'  !; !-._  __  _
      (_,         .-:'  !; !-._
    ,'o"(        (_,           )
   (__,-'      ,'o"(            )>
      (       (__,-'            )
       !-'._.--._(             )
          |||  |||!-'._.--._.-'
                     |||  |||   (Artist: Bob Allison)

`, "!", "`")
}

func New() *cobra.Command {
	root := cli.Command(&Herd{}, cobra.Command{
		Long: "\n   Herd" + sheep() + "Building cute fluffy apps since 2022.",
	})
	root.AddCommand(
		NewBuild(),
	)
	return root
}

type Herd struct {
	OptionOne string `usage:"Some usage description"`
	OptionTwo string `name:"custom-name"`
}

func (a *Herd) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
