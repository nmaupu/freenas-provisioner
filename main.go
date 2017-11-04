package main

import (
	"github.com/nmaupu/freenas-provisioner/freenas"
	"log"
	"os"
)

func main() {
	server := freenas.NewFreenasServer(os.Getenv("URL"), os.Getenv("USER"), os.Getenv("PASS"), true)
	var err error

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
		log.Fatalf("Dataset cannot be created, %v", err)
	}

	err = ds.Get(server)
	if err != nil {
		log.Fatalf("%v, ds=%+v", err, ds)
	}
	log.Printf("%+v", ds)

	// Nfs share creation
	err = nfs.Create(server)
	if err != nil {
		log.Fatalf("NfsShare cannot be created, %v", err)
	}

	err = nfs.Get(server)
	if err != nil {
		log.Fatalf("%v, nfs=%+v", err, nfs)
	}
	log.Printf("%+v", nfs)

	// Deletion
	err = nfs.Delete(server)
	if err != nil {
		log.Fatalf("%v", err)
	}

	err = ds.Delete(server)
	if err != nil {
		log.Fatalf("%v", err)
	}
}
