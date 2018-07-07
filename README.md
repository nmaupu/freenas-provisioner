[![Build Status](https://travis-ci.org/nmaupu/freenas-provisioner.svg?branch=master)](https://travis-ci.org/nmaupu/freenas-provisioner)
[![Go Report Card](https://goreportcard.com/badge/github.com/nmaupu/freenas-provisioner)](https://goreportcard.com/report/github.com/nmaupu/freenas-provisioner)

# What is freenas-provisioner
FreeNAS-provisioner is a Kubernetes external provisioner.
When a `PersisitentVolumeClaim` appears on a Kube cluster, the provisioner will
make the corresponding calls to the configured FreeNAS API to create a dataset
and a NFS share usable by the claim. When the claim or the persistent volume is
deleted, the provisioner deletes the previously created dataset and share.

See this for more info on external provisioner:
https://github.com/kubernetes-incubator/external-storage

# Usage
The scope of the provisioner allows for a single instance to service multiple
classes (and/or FreeNAS servers).  The provisioner itself can be deployed into
the cluster or ran directly on a FreeNAS server.

Each `StorageClass` should have a corresponding `Secret` created which contains
the credentials and host information used to communicate with with FreeNAS API.
In essence each `Secret` corresponds to a FreeNAS server.

The `Secret` namespace and name may be customized using the appropriate
`StorageClass` `parameters`.  By default `kube-system` and `freenas-nfs` are
used.  While multiple `StorageClass` resources may point to the same server
and hence same `Secret`, it is recommended to create a new `Secret` for each
`StorageClass` resource.

It is **highly** recommended to read `deploy/claim.yaml` to review available
`parameters` and gain a better understanding of functionality and behavior.

## FreeNAS Setup
You must manually create a dataset.  You may simply use a pool as the parent
dataset but it's recommended to create a dedicated dataset.

Additionally, you need to enabled the NFS service.  It's highly recommended to
configure the NFS service as v3.  If v4 must be used then it's also recommended
to enable the `NFSv3 ownership model for NFSv4` option.

## Provision the provisioner
Run it on the cluster:
```
kubectl apply -f deploy/rbac.yaml -f deploy/deployment.yaml
```

Alternatively, for advanced use-cases you may run the provisioner out of cluster
directly on the FreeNAS server.  This is not currently recommended.
```
./bin/freenas-provisioner-freebsd --kubeconfig=/path/to/kubeconfig.yaml
```

## Create `StorageClass` and `Secret`
All the necessary resources are available in the `deploy` folder.  At a minimum
`secret.yaml` must be modified (remember to `base64` the values) to reflect the
server details.  You may also want to review `class.yaml` to review available
`parameters` of the storage class.  For instance to set the `datasetParentName`.
```
kubectl apply -f deploy/class.yaml -f deploy/secret.yaml
```

## Example usage
Next, create a `PersistentVolumeClaim` using the storage class
(`deploy/test-claim.yaml`):
```
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: freenas-test-pvc
spec:
  storageClassName: freenas-nfs
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Mi
```

Use that claim on a testing pod (`deploy/test-pod.yaml`):
```
---
kind: Pod
apiVersion: v1
metadata:
  name: freenas-test-pod
spec:
  containers:
  - name: freenas-test-pod
    image: gcr.io/google_containers/busybox:1.24
    command:
      - "/bin/sh"
    args:
      - "-c"
      - "date >> /mnt/file.log && exit 0 || exit 1"
    volumeMounts:
      - name: freenas-test-volume
        mountPath: "/mnt"
  restartPolicy: "Never"
  volumes:
    - name: freenas-test-volume
      persistentVolumeClaim:
        claimName: freenas-test-pvc
```

Create everything:
```
kubectl apply -f deploy/
```

The underlying dataset / NFS share should quickly be appearing up on FreeNAS
side.  In case of issue, follow the provisioner's logs using:
```
kubectl -n kube-system logs -f freenas-nfs-provisioner-<id>
```

# Development
```
make vendor && make
```
Binary is located into `bin/freenas-provisioner`.  It is compiled to be run on
`linux-amd64` by default, but you may run the following for different builds:
```
make vendor && make darwin
# OR
make vendor && make freebsd
```

To run locally with an appropriate `$KUBECONFIG` you may run:
```
./local-start.sh
```

To format code before committing:
```
make fmt
```

## Docs
 * https://github.com/kubernetes/community/tree/master/contributors/design-proposals/storage
 * https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/volume-provisioning.md
 * https://kubernetes.io/docs/concepts/storage/storage-classes/
 * https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#-strong-api-overview-strong-

## TODO
 * volume resizing - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/grow-volume-size.md
 * volume snapshots - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/volume-snapshotting.md
 * mount options - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/mount-options.md
 * ~~support multiple instances (secrets in storage class)~~
 * cleanup empty namespaces?
 * ~~do not delete when deterministic volumes pre-existed (ie: only delete if the provisioner created volume)~~
  * https://github.com/kubernetes-incubator/external-storage/blob/master/ceph/cephfs/cephfs-provisioner.go#L225
 * iscsi

## Notes
To sniff API traffic between host and server:
```
sudo tcpdump -A -s 0 'host <server ip> and tcp port 80 and (((ip[2:2] - ((ip[0]&0xf)<<2)) - ((tcp[12]&0xf0)>>2)) != 0)'
```
