package provisioner

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/nmaupu/freenas-provisioner/freenas"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"path/filepath"
)

var (
	// freenasProvisioner is an implem of controller.Provisioner
	_ controller.Provisioner = &freenasProvisioner{}
)

type freenasProvisioner struct {
	Pool, Mountpoint, ParentDataset string
	Identifier                      string
	FreenasServer                   *freenas.FreenasServer
}

func New(pool, mountpoint, parentDataset, identifier string, freenasServer *freenas.FreenasServer) controller.Provisioner {
	return &freenasProvisioner{
		Pool:          pool,
		Mountpoint:    mountpoint,
		ParentDataset: parentDataset,
		Identifier:    identifier,
		FreenasServer: freenasServer,
	}
}

func (p *freenasProvisioner) getMountpoint() string {
	return filepath.Join(p.Mountpoint, p.Pool, p.ParentDataset)
}

// Provision a dataset and creates an NFS share on Freenas side
func (p *freenasProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	path := filepath.Join(p.getMountpoint(), options.PVName)
	glog.Infof("Provisioning %s", path)

	ds := freenas.Dataset{
		Pool: p.Pool,
		Name: filepath.Join(p.ParentDataset, options.PVName),
	}

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
	err = ds.Create(p.FreenasServer)
	if err != nil {
		return nil, err
	}
	err = share.Create(p.FreenasServer)
	if err != nil {
		return nil, err
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
			Annotations: map[string]string{
				"freenasProvisionerIdentity": p.Identifier,
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
					Server:   p.FreenasServer.Host,
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

	share := freenas.NfsShare{
		Paths: []string{path},
	}
	ds := freenas.Dataset{
		Pool: p.Pool,
		Name: filepath.Join(p.ParentDataset, pvName),
	}
	glog.Infof("Deleting dataset: %s/%s, NFS share: %s", ds.Pool, ds.Name, path)

	err = share.Get(p.FreenasServer)
	if err != nil {
		glog.Warningf(fmt.Sprintf("Could not find NFS share %s on server side, already deleted ?", path))
	} else {
		err = share.Delete(p.FreenasServer)
		if err != nil {
			return errors.New(fmt.Sprintf("Could not delete NFS share %s on server side, ignoring. Error: %v", path, err))
		}
	}

	err = ds.Get(p.FreenasServer)
	if err != nil {
		glog.Warningf(fmt.Sprintf("Could not find dataset %s on server side, already deleted ?", ds))
	} else {
		err = ds.Delete(p.FreenasServer)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot delete dataset %s. Error: %v", ds, err))
		}
	}

	return nil
}
