package provisioner

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/nmaupu/freenas-provisioner/freenas"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	// freenasProvisioner is an implem of controller.Provisioner
	_ controller.Provisioner = &freenasProvisioner{}
)

type freenasProvisionerConfig struct {
	// Dataset options
	DatasetParentName               string
	DatasetEnableQuotas             bool
	DatasetEnableReservation        bool
	DatasetEnableNamespaces         bool
	DatasetEnableDeterministicNames bool
	DatasetRetainPreExisting        bool
	DatasetPermissionsMode          string
	DatasetPermissionsUser          string
	DatasetPermissionsGroup         string

	// Share options
	ShareHost              string
	ShareAlldirs           bool
	ShareAllowedHosts      string
	ShareAllowedNetworks   string
	ShareMaprootUser       string
	ShareMaprootGroup      string
	ShareMapallUser        string
	ShareMapallGroup       string
	ShareRetainPreExisting bool

	// Server options
	ServerSecretNamespace string
	ServerSecretName      string
	ServerProtocol        string
	ServerHost            string
	ServerPort            int
	ServerUsername        string
	ServerPassword        string
	ServerAllowInsecure   bool
}

func (p *freenasProvisioner) GetConfig(storageClassName string) (*freenasProvisionerConfig, error) {
	class, err := p.Client.StorageV1beta1().StorageClasses().Get(storageClassName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// dataset defaults
	var datasetParentName string = "tank"
	var datasetEnableQuotas bool = true
	var datasetEnableReservation bool = true
	var datasetEnableNamespaces bool = true
	var datasetEnableDeterministicNames bool = true
	var datasetRetainPreExisting bool = true
	var datasetPermissionsMode string = "0777"
	var datasetPermissionsUser string = "root"
	var datasetPermissionsGroup string = "wheel"

	// share defaults
	var shareHost string = ""
	var shareAlldirs bool = true
	var shareAllowedHosts string = ""
	var shareAllowedNetworks string = ""
	var shareMaprootUser string = "root"
	var shareMaprootGroup string = "wheel"
	var shareMapallUser string = ""
	var shareMapallGroup string = ""
	var shareRetainPreExisting bool = true

	// server options
	var serverSecretNamespace string = "kube-system"
	var serverSecretName string = "freenas-nfs"
	var serverProtocol string = "http"
	var serverHost string = "localhost"
	var serverPort int = 80
	var serverUsername string = "root"
	var serverPassword string = ""
	var serverAllowInsecure bool = false

	// set values from StorageClass parameters
	for k, v := range class.Parameters {
		switch k {
		// Dataset options
		case "datasetParentName":
			datasetParentName = v
		case "datasetEnableQuotas":
			datasetEnableQuotas, _ = strconv.ParseBool(v)
		case "datasetEnableReservation":
			datasetEnableReservation, _ = strconv.ParseBool(v)
		case "datasetEnableNamespaces":
			datasetEnableNamespaces, _ = strconv.ParseBool(v)
		case "datasetEnableDeterministicNames":
			datasetEnableDeterministicNames, _ = strconv.ParseBool(v)
		case "datasetRetainPreExisting":
			datasetRetainPreExisting, _ = strconv.ParseBool(v)
		case "datasetPermissionsMode":
			datasetPermissionsMode = v
		case "datasetPermissionsUser":
			datasetPermissionsUser = v
		case "datasetPermissionsGroup":
			datasetPermissionsGroup = v

		// Share options
		case "shareHost":
			shareHost = v
		case "shareAlldirs":
			shareAlldirs, _ = strconv.ParseBool(v)
		case "shareAllowedHosts":
			shareAllowedHosts = v
		case "shareAllowedNetworks":
			shareAllowedNetworks = v
		case "shareMaprootUser":
			shareMaprootUser = v
		case "shareMaprootGroup":
			shareMaprootGroup = v
		case "shareMappallUser":
			shareMapallUser = v
		case "shareMapallGroup":
			shareMapallGroup = v
		case "shareRetainPreExisting":
			shareRetainPreExisting, _ = strconv.ParseBool(v)

		// Server options
		case "serverSecretNamespace":
			serverSecretNamespace = v
		case "serverSecretName":
			serverSecretName = v
		}
	}

	secret, err := p.GetSecret(serverSecretNamespace, serverSecretName)
	if err != nil {
		return nil, err
	}

	// set values from secret
	for k, v := range secret.Data {
		switch k {
		case "protocol":
			serverProtocol = BytesToString(v)
		case "host":
			serverHost = BytesToString(v)
		case "port":
			serverPort, _ = strconv.Atoi(BytesToString(v))
		case "username":
			serverUsername = BytesToString(v)
		case "password":
			serverPassword = BytesToString(v)
		case "allowInsecure":
			serverAllowInsecure, _ = strconv.ParseBool(BytesToString(v))
		}
	}

	if shareHost == "" {
		shareHost = serverHost
	}

	return &freenasProvisionerConfig{
		// Dataset options
		DatasetParentName:               datasetParentName,
		DatasetEnableQuotas:             datasetEnableQuotas,
		DatasetEnableReservation:        datasetEnableReservation,
		DatasetEnableNamespaces:         datasetEnableNamespaces,
		DatasetEnableDeterministicNames: datasetEnableDeterministicNames,
		DatasetRetainPreExisting:        datasetRetainPreExisting,
		DatasetPermissionsMode:          datasetPermissionsMode,
		DatasetPermissionsUser:          datasetPermissionsUser,
		DatasetPermissionsGroup:         datasetPermissionsGroup,

		// Share options
		ShareHost:              shareHost,
		ShareAlldirs:           shareAlldirs,
		ShareAllowedHosts:      shareAllowedHosts,
		ShareAllowedNetworks:   shareAllowedNetworks,
		ShareMaprootUser:       shareMaprootUser,
		ShareMaprootGroup:      shareMaprootGroup,
		ShareMapallUser:        shareMapallUser,
		ShareMapallGroup:       shareMapallGroup,
		ShareRetainPreExisting: shareRetainPreExisting,

		// Server options
		ServerSecretNamespace: serverSecretNamespace,
		ServerSecretName:      serverSecretName,
		ServerProtocol:        serverProtocol,
		ServerHost:            serverHost,
		ServerPort:            serverPort,
		ServerUsername:        serverUsername,
		ServerPassword:        serverPassword,
		ServerAllowInsecure:   serverAllowInsecure,
	}, nil
}

type freenasProvisioner struct {
	Client     kubernetes.Interface
	Identifier string
}

func New(client kubernetes.Interface, identifier string) controller.Provisioner {
	return &freenasProvisioner{
		Client:     client,
		Identifier: identifier,
	}
}

// Provision a dataset and creates an NFS share on Freenas side
func (p *freenasProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	var err error

	// get config
	config, err := p.GetConfig(*options.PVC.Spec.StorageClassName)
	if err != nil {
		return nil, err
	}
	//glog.Infof("%+v\n", config)

	// get server
	freenasServer, err := p.GetServer(*config)
	if err != nil {
		return nil, err
	}

	// get parent dataset
	parentDs := freenas.Dataset{
		Name: config.DatasetParentName,
	}
	err = parentDs.Get(freenasServer)
	if err != nil {
		return nil, err
	}

	meta := options.PVC.GetObjectMeta()
	dsName := options.PVName
	dsNamespace := ""

	if config.DatasetEnableNamespaces {
		dsNamespace = meta.GetNamespace()
	}

	if config.DatasetEnableDeterministicNames {
		if config.DatasetEnableNamespaces {
			dsName = meta.GetName()
		} else {
			dsName = meta.GetNamespace() + "-" + meta.GetName()
		}
	}

	path := filepath.Join(parentDs.Mountpoint, dsNamespace, dsName)
	dsPath := filepath.Join(config.DatasetParentName, dsNamespace, dsName)
	datasetComments := fmt.Sprintf("%s/%s/%s", meta.GetClusterName(), meta.GetNamespace(), meta.GetName())
	var datasetRefquota, datasetRefreservation, datasetRecordsize int64 = 0, 0, 0

	if config.DatasetEnableQuotas {
		volSize := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
		datasetRefquota = volSize.Value()
	}

	if config.DatasetEnableReservation {
		volSize := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
		datasetRefreservation = volSize.Value()
	}

	ds := freenas.Dataset{
		Pool:           parentDs.Pool,
		Name:           dsPath,
		Refquota:       datasetRefquota,
		Refreservation: datasetRefreservation,
		Recordsize:     datasetRecordsize,
		Comments:       datasetComments,
	}

	share := freenas.NfsShare{
		Paths:        []string{path},
		ReadOnly:     false,
		Alldirs:      config.ShareAlldirs,
		Hosts:        config.ShareAllowedHosts,
		Network:      config.ShareAllowedNetworks,
		MapallUser:   config.ShareMapallUser,
		MapallGroup:  config.ShareMapallGroup,
		MaprootUser:  config.ShareMaprootUser,
		MaprootGroup: config.ShareMaprootGroup,
		Comment:      fmt.Sprintf("freenas-provisioner (%s): %s", p.Identifier, dsPath),
	}

	glog.Infof("Creating dataset: \"%s\", NFS share: \"%s\"", ds.Name, path)

	// Provisioning dataset and nfs share
	var datasetPreExisted, sharePreExisted = false, false
	if config.DatasetEnableNamespaces {
		nsDs := freenas.Dataset{
			Pool:     parentDs.Pool,
			Name:     filepath.Join(parentDs.Name, dsNamespace),
			Comments: "k8s provisioned namespace",
		}

		err = nsDs.Get(freenasServer)
		if err != nil {
			glog.Infof("creating namespace dataset \"%s\"", nsDs.Name)
			err = nsDs.Create(freenasServer)
		} else {
			glog.Infof("namespace dataset \"%s\" already exists", nsDs.Name)
		}
	}
	if err != nil {
		return nil, err
	}

	if config.DatasetEnableDeterministicNames {
		err = ds.Get(freenasServer)

		if err != nil {
			err = ds.Create(freenasServer)
		} else {
			datasetPreExisted = true
			glog.Infof("dataset \"%s\" already exists", ds.Name)
		}
	} else {
		err = ds.Create(freenasServer)
	}
	if err != nil {
		return nil, err
	}

	if config.DatasetEnableDeterministicNames {
		err = share.Get(freenasServer)
		if err != nil {
			err = share.Create(freenasServer)
		} else {
			sharePreExisted = true
			glog.Infof("share \"%s\" already exists", path)
		}
	} else {
		err = share.Create(freenasServer)
	}
	if err != nil {
		return nil, err
	}

	glog.Infof("setting permissions on path \"%s\" to - mode: %s, owner: %s:%s", path, config.DatasetPermissionsMode, config.DatasetPermissionsUser, config.DatasetPermissionsGroup)
	permission := freenas.Permission{
		Path:  path,
		Acl:   "unix",
		Mode:  config.DatasetPermissionsMode,
		User:  config.DatasetPermissionsUser,
		Group: config.DatasetPermissionsGroup,
	}
	err = permission.Put(freenasServer)
	if err != nil {
		return nil, err
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
			Annotations: map[string]string{
				"freenasNFSProvisionerIdentity": p.Identifier,
				"datasetPreExisted":             strconv.FormatBool(datasetPreExisted),
				"sharePreExisted":               strconv.FormatBool(sharePreExisted),
				"shareId":                       strconv.Itoa(share.Id),
				"datasetEnableQuotas":           strconv.FormatBool(config.DatasetEnableQuotas),
				"datasetEnableReservation":      strconv.FormatBool(config.DatasetEnableReservation),
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
					Server:   config.ShareHost,
					Path:     path,
					ReadOnly: false,
				},
			},
		},
	}

	return pv, nil
}

// Prep for resizing
// https://github.com/kubernetes-incubator/external-storage/blob/master/gluster/file/cmd/glusterfile-provisioner/glusterfile-provisioner.go#L433
/*
func (p *freenasProvisioner) RequiresFSResize() bool {
	return false
}

func (p *glusterfileProvisioner) ExpandVolumeDevice(spec *volume.Spec, newSize resource.Quantity, oldSize resource.Quantity) (resource.Quantity, error) {
	return newVolumeSize, nil
}
*/

func (p *freenasProvisioner) Delete(volume *v1.PersistentVolume) error {
	var datasetPreExisted, sharePreExisted bool = false, false
	var shareId int

	shareIdAnnotation, ok := volume.Annotations["shareId"]
	if ok {
		shareId, _ = strconv.Atoi(shareIdAnnotation)
	}

	datasetPreExistedAnnotation, ok := volume.Annotations["datasetPreExisted"]
	if ok {
		datasetPreExisted, _ = strconv.ParseBool(datasetPreExistedAnnotation)
	}

	sharePreExistedAnnotation, ok := volume.Annotations["sharePreExisted"]
	if ok {
		sharePreExisted, _ = strconv.ParseBool(sharePreExistedAnnotation)
	}

	var err error

	// get config
	config, err := p.GetConfig(volume.Spec.StorageClassName)
	if err != nil {
		return err
	}
	//glog.Infof("%+v\n", config)

	// get server
	freenasServer, err := p.GetServer(*config)
	if err != nil {
		return err
	}

	// get parent dataset
	parentDs := freenas.Dataset{
		Name: config.DatasetParentName,
	}
	err = parentDs.Get(freenasServer)
	if err != nil {
		return err
	}

	// hydrate share
	path := volume.Spec.PersistentVolumeSource.NFS.Path
	share := freenas.NfsShare{
		Id:    shareId,
		Paths: []string{path},
	}

	// hydrate dataset
	ds := freenas.Dataset{
		Pool: parentDs.Pool,
		Name: config.DatasetParentName + strings.SplitN(path, config.DatasetParentName, 2)[1],
	}
	glog.Infof("Deleting dataset: \"%s\", NFS share: \"%s\"", ds.Name, path)

	// delete share
	if (sharePreExisted == true && !config.ShareRetainPreExisting) || !sharePreExisted {
		err = share.Get(freenasServer)
		if err != nil {
			glog.Warningf(fmt.Sprintf("Could not find NFS share \"%s\" on server side, already deleted?", path))
		} else {
			err = share.Delete(freenasServer)
			if err != nil {
				return errors.New(fmt.Sprintf("Could not delete NFS share \"%s\" on server side, ignoring. Error: %v", path, err))
			}
		}
	}

	// delete dataset
	if (datasetPreExisted == true && !config.DatasetRetainPreExisting) || !datasetPreExisted {
		err = ds.Get(freenasServer)
		if err != nil {
			glog.Warningf(fmt.Sprintf("Could not find dataset \"%s\" on server side, already deleted ?", ds.Name))
		} else {
			err = ds.Delete(freenasServer)
			if err != nil {
				return errors.New(fmt.Sprintf("Cannot delete dataset \"%s\". Error: %v", ds.Name, err))
			}
		}
	}

	return nil
}

func (p *freenasProvisioner) GetServer(config freenasProvisionerConfig) (*freenas.FreenasServer, error) {
	return freenas.NewFreenasServer(
		config.ServerProtocol, config.ServerHost, config.ServerPort,
		config.ServerUsername, config.ServerPassword,
		config.ServerAllowInsecure,
	), nil
}

func (p *freenasProvisioner) GetSecret(namespace, secretName string) (*v1.Secret, error) {
	if p.Client == nil {
		return nil, fmt.Errorf("Cannot get kube client")
	}
	return p.Client.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
}

func BytesToString(data []byte) string {
	return string(data[:])
}
