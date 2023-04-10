package cli

import (
	"fmt"
	"os"
	"path/filepath"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func NewMan(c CommandContext) *cobra.Command {
	return cli.Command(&Man{client: c.ClientFactory}, cobra.Command{
		Use: "man",
		Example: `
acorn man`,
		SilenceUsage: true,
		Short:        "Generate acorn man pages",
		Hidden:       true,
	})
}

type Man struct {
	Directory string `usage:"Directory to generate man files" short:"d"`
	client    ClientFactory
}

func (m *Man) Run(cmd *cobra.Command, args []string) error {
	// Create root command to create pages from
	rootCmd := New()

	var manDir string
	if m.Directory == "" {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("Could not find executable directory: %v\n", err)
		}

		// Get the directory containing the binary file
		exeDir := filepath.Dir(exePath)

		// Determine the directory to place the manual files in
		manDir = filepath.Join(exeDir, "manpages")
		// Create subfolders if they don't exist
		fmt.Printf("creating %v", manDir)
		err = os.MkdirAll(manDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("Could not generate necessary directories: %v\n", err)
		}
	} else {
		manDir = m.Directory
	}

	//file, err := os.OpenFile(filepath.Join(manDir, "acorn.txt"), os.O_TRUNC, 0777)
	//defer file.Close()
	//if err != nil {
	//	return fmt.Errorf("Can't open file for writing: %v\n", err)
	//}
	//err = doc.GenMan(rootCmd, &doc.GenManHeader{
	//	Title:   "Acorn",
	//	Section: "",
	//	Date:    nil,
	//	Source:  "docs.acorn.io",
	//	Manual:  "",
	//},
	//	file)
	//if err != nil {
	//	return fmt.Errorf("Can't open file for writing: %v\n", err)
	//}

	err := doc.GenManTree(rootCmd, &doc.GenManHeader{
		Title:  "Acorn",
		Manual: "Acorn Docs",
		Source: "docs.acorn.io",
	}, manDir)
	if err != nil {
		return fmt.Errorf("Error generating manual pages for acorn in installation directory: %v\n", err)
	}
	return nil
}
