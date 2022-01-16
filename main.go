package main

import (
	"github.com/ibuildthecloud/herd/pkg/herd"
	cli "github.com/rancher/wrangler-cli"
)

func main() {
	cli.Main(herd.New())
}
