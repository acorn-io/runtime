package main

import (
	herd "github.com/ibuildthecloud/herd/pkg/cli"
	cli "github.com/rancher/wrangler-cli"
)

func main() {
	cli.Main(herd.New())
}
