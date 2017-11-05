package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/nmaupu/freenas-provisioner/freenas"
	"github.com/nmaupu/freenas-provisioner/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	provisionerName           = "maupu.org/freenas"
	exponentialBackOffOnError = false
	failedRetryThreshold      = 5
	leasePeriod               = controller.DefaultLeaseDuration
	retryPeriod               = controller.DefaultRetryPeriod
	renewDeadline             = controller.DefaultRenewDeadline
	termLimit                 = controller.DefaultTermLimit
)

var (
	_      controller.Provisioner = &freenasProvisioner{}
	server *freenas.FreenasServer
)

type freenasProvisioner struct {
	identity string
}

func NewFreenasProvisioner() controller.Provisioner {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		glog.Fatalf("env var NODE_NAME must be set so that this provisioner can identify itself")
	}

	return &freenasProvisioner{
		identity: nodeName,
	}
}

func (p *freenasProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	logging.GetLogger().Infof("Provisioning tank/%s", options.PVName)
	ds := freenas.Dataset{
		Pool: "tank",
		Name: options.PVName,
	}

	path := filepath.Join("/mnt/tank", options.PVName)
	share := freenas.NfsShare{
		Paths:       []string{path},
		ReadOnly:    false,
		Alldirs:     true,
		Hosts:       "knode1 knode2 knode3",
		MapallUser:  "root",
		MapallGroup: "wheel",
		Comment:     "Created from freenas-provisioner",
	}

	// Provisioning dataset and nfs share
	var err error
	err = ds.Create(server)
	if err != nil {
		return nil, err
	}
	err = share.Create(server)
	if err != nil {
		return nil, err
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
			Annotations: map[string]string{
				"freenasProvisionerIdentity": p.identity,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					Server:   server.Host,
					Path:     path,
					ReadOnly: false,
				},
			},
		},
	}

	return pv, nil
}

func (p *freenasProvisioner) Delete(volume *v1.PersistentVolume) error {
	var err error
	path := volume.Spec.PersistentVolumeSource.NFS.Path
	pvName := filepath.Base(path)
	logging.GetLogger().Infof("Deleting tank/%s", pvName)

	share := freenas.NfsShare{
		Paths: []string{path},
	}

	err = share.Get(server)
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot find %s volume on server side", path))
	}

	err = share.Delete(server)
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot delete share %+v", share))
	}

	ds := freenas.Dataset{
		Pool: "tank",
		Name: pvName,
	}
	err = ds.Delete(server)
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot delete dataset %+v", ds))
	}

	return nil
}

func main() {
	var err error

	logging.GetLogger().Infoln("Starting freenas-provisioner")

	syscall.Umask(0)

	flag.Parse()
	flag.Set("logtostderr", "true")

	host := os.Getenv("FREENAS_HOST")
	if url == "" {
		glog.Fatal("FREENAS_URL not set")
	}

	port := os.Getenv("FREENAS_PORT")
	if port == "" {
		glog.Fatal("FREENAS_PORT")
	}

	user := os.Getenv("FREENAS_USER")
	if user == "" {
		glog.Fatal("FREENAS_USER not set")
	}

	pass := os.Getenv("FREENAS_PASSWORD")
	if pass == "" {
		glog.Fatal("FREENAS_PASSWORD not set")
	}

	server = freenas.NewFreenasServer(host, port, user, pass, true)

	// Create an InClusterConfig and use it to create a client for the controller
	// to use to communicate with Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	clientFreenasProvisioner := NewFreenasProvisioner()

	// Start the provision controller which will dynamically provision datasets and nfs shares
	pc := controller.NewProvisionController(
		clientset,
		15*time.Second,
		provisionerName,
		clientFreenasProvisioner,
		serverVersion.GitVersion,
		exponentialBackOffOnError,
		failedRetryThreshold,
		leasePeriod,
		renewDeadline,
		retryPeriod,
		termLimit,
	)
	pc.Run(wait.NeverStop)
}
