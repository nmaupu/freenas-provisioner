package main

import (
	"github.com/nmaupu/freenas-provisioner/cli"
)

const (
	AppName = "freenas-provisioner"
	AppDesc = "Kubernetes Freenas Provisioner (NFS)"
)

var (
	AppVersion string
)

func main() {
	if AppVersion == "" {
		AppVersion = "master"
	}

	cli.Process(AppName, AppDesc, AppVersion)
}
