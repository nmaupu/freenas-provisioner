package provisioner

import (
	"errors"
	"fmt"
	"strings"
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
	meta := options.PVC.GetObjectMeta()
	deterministicNames := false
	datasetNamespaces := false
	dsName := options.PVName
	dsNamespace := ""
	if (strings.ToLower(options.Parameters["datasetNamespaces"]) == "true" ) {
		datasetNamespaces = true
		dsNamespace = meta.GetNamespace()
	}

	if (strings.ToLower(options.Parameters["datasetDeterministicNames"]) == "true" ) {
		deterministicNames = true
		if (datasetNamespaces) {
			dsName = meta.GetName()
		} else {
			dsName = meta.GetNamespace() + "-" + meta.GetName()
		}
	}

	path := filepath.Join(p.getMountpoint(), dsNamespace, dsName)
	glog.Infof("Provisioning %s", path)

	datasetComments := fmt.Sprintf("%s/%s/%s", meta.GetClusterName(), meta.GetNamespace(), meta.GetName())

	var datasetRefquota, datasetRefreservation, datasetRecordsize int64 = 0, 0, 0;
	var shareHosts, shareNetwork, shareMapallUser, shareMapallGroup, shareMaprootUser, shareMaprootGroup string = "", "", "root", "wheel", "", ""
	var shareAlldirs bool = true

	for k, v := range options.Parameters {
		switch k {
		case "datasetEnableQuotas":
			if (strings.ToLower(v) == "true") {
				volSize := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
				datasetRefquota = volSize.Value()
			}
		case "datasetEnableReservation":
			if (strings.ToLower(v) == "true") {
				volSize := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
				datasetRefreservation = volSize.Value()
			}
		case "shareHosts":
			shareHosts = v
		case "shareNetwork":
			shareNetwork = v
		case "shareMappallUser":
			shareMapallUser = v
		case "shareMapallGroup":
			shareMapallGroup = v
		case "shareMaprootUser":
			shareMaprootUser = v
		case "shareMaprootGroup":
			shareMaprootGroup = v
		case "shareAlldirs":
			if (strings.ToLower(v) == "true") {
				shareAlldirs = true
			} else {
				shareAlldirs = false
			}
		}
	}

	dsPath := filepath.Join(p.Pool, p.ParentDataset, dsNamespace, dsName)
	ds := freenas.Dataset{
		Pool: p.Pool,
		Name: dsPath,
		Refquota: datasetRefquota,
		Refreservation: datasetRefreservation,
		Recordsize: datasetRecordsize,
		Comments: datasetComments,
	}

	share := freenas.NfsShare{
		Paths:        []string{path},
		ReadOnly:     false,
		Alldirs:      shareAlldirs,
		Hosts:        shareHosts,
		Network:      shareNetwork,
		MapallUser:   shareMapallUser,
		MapallGroup:  shareMapallGroup,
		MaprootUser:  shareMaprootUser,
		MaprootGroup: shareMaprootGroup,
		Comment:      fmt.Sprintf("freenas-provisioner (%s): %s", p.Identifier, dsName),
	}

	// Provisioning dataset and nfs share
	var err error
	if (datasetNamespaces) {
		nsDs := freenas.Dataset{
			Pool: p.Pool,
			Name: filepath.Join(p.Pool, p.ParentDataset, dsNamespace),
		}

		err = nsDs.Get(p.FreenasServer)
		if err != nil {
			glog.Infof("creating namespace dataset %s", nsDs.Name)
			err = nsDs.Create(p.FreenasServer)
		} else {
			glog.Infof("namespace dataset already exists %s", nsDs.Name)
		}
	}
	if err != nil {
		return nil, err
	}

	if (deterministicNames) {
		err = ds.Get(p.FreenasServer)

		if err != nil {
			err = ds.Create(p.FreenasServer)
		} else {
			glog.Infof("dataset already exists %s", ds.Name)
		}
	} else {
		err = ds.Create(p.FreenasServer)
	}
	if err != nil {
		return nil, err
	}

	if (deterministicNames) {
		err = share.Get(p.FreenasServer)
		if err != nil {
			err = share.Create(p.FreenasServer)
		} else {
			glog.Infof("share already exists %s", path)
		}
	} else {
		err = share.Create(p.FreenasServer)
	}
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

	share := freenas.NfsShare{
		Paths: []string{path},
	}
	ds := freenas.Dataset{
		Pool: p.Pool,
		Name: strings.TrimPrefix(path, p.Mountpoint + "/"),
	}
	glog.Infof("Deleting dataset: %s, NFS share: %s", ds.Name, path)

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
		glog.Warningf(fmt.Sprintf("Could not find dataset %v on server side, already deleted ?", ds))
	} else {
		err = ds.Delete(p.FreenasServer)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot delete dataset %v. Error: %v", ds, err))
		}
	}

	return nil
}
