package main

import (
	"io"
	"log"
	"os"

	"github.com/ibuildthecloud/herd/pkg/appimageinit"
)

func copyTarget(dst string) error {
	src := os.Args[0]
	if err := os.Link(src, dst); err == nil {
		return nil
	}

	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "init" {
		if err := copyTarget(os.Args[2]); err != nil {
			log.Fatal(err)
		}
		return
	}

	output := os.Getenv("OUTPUT")
	id := os.Getenv("ID")

	wr, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer wr.Close()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if err := appimageinit.Print(cwd, id, wr); err != nil {
		wr.WriteString(err.Error())
	}
}
