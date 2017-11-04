package main

import (
	"github.com/nmaupu/freenas-provisioner/freenas"
	"github.com/nmaupu/freenas-provisioner/logging"
	"os"
)

func main() {
	server := freenas.NewFreenasServer(os.Getenv("URL"), os.Getenv("USER"), os.Getenv("PASS"), true)
	var err error
	log := logging.GetLogger()

	ds := freenas.Dataset{
		Pool: "tank",
		Name: "gotest",
	}
	nfs := freenas.NfsShare{
		Paths:       []string{"/mnt/tank/gotest"},
		ReadOnly:    true,
		Alldirs:     true,
		MapallUser:  "root",
		MapallGroup: "wheel",
		Hosts:       "knode1 knode2 knode3",
	}

	// Dataset creation

	err = ds.Create(server)
	if err != nil {
		log.Warnf("Dataset cannot be created, %v", err)
	}

	err = ds.Get(server)
	if err != nil {
		log.Warnf("%v, ds=%+v", err, ds)
	}
	log.Infof("%+v", ds)

	// Nfs share creation
	err = nfs.Create(server)
	if err != nil {
		log.Warnf("NfsShare cannot be created, %v", err)
	}

	err = nfs.Get(server)
	if err != nil {
		log.Warnf("%v, nfs=%+v", err, nfs)
	}
	log.Infof("%+v", nfs)

	// Deletion
	err = nfs.Delete(server)
	if err != nil {
		log.Warnf("%v", err)
	}

	err = ds.Delete(server)
	if err != nil {
		log.Warnf("%v", err)
	}
}
