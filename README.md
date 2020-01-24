[![Build Status](https://travis-ci.org/nmaupu/freenas-provisioner.svg?branch=master)](https://travis-ci.org/nmaupu/freenas-provisioner)
[![Go Report Card](https://goreportcard.com/badge/github.com/nmaupu/freenas-provisioner)](https://goreportcard.com/report/github.com/nmaupu/freenas-provisioner)

# freenas-provisioner

FreeNAS-provisioner is a Kubernetes external provisioner.
When a `PersistentVolumeClaim` appears on a Kube cluster, the provisioner will
make the corresponding calls to the configured FreeNAS API to create a dataset
and a NFS share usable by the claim. When the claim or the persistent volume is
deleted, the provisioner deletes the previously created dataset and share.

See this for more info on external provisioner:
https://github.com/kubernetes-incubator/external-storage

# Usage

The scope of the provisioner allows for a single instance to service multiple
classes (and/or FreeNAS servers).  The provisioner itself can be deployed into
the cluster or ran out of cluster, for example, directly on a FreeNAS server.

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

## Install via Helm

Using [Helm](https://helm.sh), you can easily install the FreeNAS Provisioner in a 
Kubernetes cluster by running the following:

```
helm install --generate-name \
  helm/freenas-provisioner \
  --set 'freenasConfig.host=<ip_or_hostname>' \
  --set 'storageClass.parameters.datasetParentName=<pool/dataset_name>' \
  --set 'freenasConfig.password=<root_password>
```

### Chart Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Set Pod affinity rules |
| freenasConfig.allowInsecure | bool | `false` | Allow for self-signed/untrusted certs if using https |
| freenasConfig.host | string | `"localhost"` | Host at which FreeNAS can be reached at. Set to localhost for running the provisioner out of cluster directly on FreeNAS node |
| freenasConfig.port | int | `80` | Port FreeNAS is running on. Usually 80 for http and 443 for https |
| freenasConfig.protocol | string | `"http"` | Protocol to use to access the FreeNAS API. Valid values are http or https |
| freenasConfig.secretName | string | `"freenas-nfs"` | name of the secret which contains FreeNAS server connection details |
| freenasConfig.useExistingSecret | bool | `false` | Set this to true to use your own Secret, created outside of this deployment |
| freenasConfig.username | string | `"root"` | Do not change. API is only available for root currently |
| freenasConfig.password | string | `nil` | Password for root user. NEVER set this in a file. Use --set freenasConfig.password=<your_password> instead |
| fullnameOverride | string | `""` | Overrides the Full Name of resources |
| image.pullPolicy | string | `"IfNotPresent"` | Docker image pull policy |
| image.pullSecrets | list | `[]` | Secrets to use when pulling Docker images |
| image.repository | string | `"docker.io/nmaupu/freenas-provisioner"` | Docker registry/repository to pull the image from |
| nameOverride | string | `""` | Overrides the name of resources |
| nodeSelector | object | `{}` | Node Selector configuration |
| podSecurityContext | object | `{}` | Set Pod security contexts |
| replicaCount | int | `1` | Number of pods to run |
| resources | object | `{}` | Set resource limits/requests for the Pod(s) |
| securityContext | object | `{}` | Set Security Context |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `nil` | name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| storageClass.annotations | object | `{}` | (Optional) annotations to add to the StorageClass. ie. `storageclass.kubernetes.io/is-default-class: "true"` |
| storageClass.reclaimPolicy | string | `"Delete"` | Reclaim Policy to use for created PVCs |
| storageClass.serverSecretNamespace | string | `nil` | Overrides the namespace of the secret which contains the FreeNAS server connection details. Default is this Release's Namespace |
| storageClass.parameters.datasetParentName | string | `"tank"` | the name of the parent dataset (or simply pool) where all resources will be created, it *must* exist before provisioner will work. ie tank/k8s/mycluster |
| storageClass.parameters.datasetEnableQuotas | string | `"true"` | whether to enforce quotas for each dataset. If enabled each newly provisioned dataset will set the appropriate quota per the PVC |
| storageClass.parameters.datasetEnableReservation | string | `"true"` | whether to reserve space when the dataset is created. If enabled each newly provisioned dataset will set the appropriate value per the PVC |
| storageClass.parameters.datasetEnableNamespaces | string | `"true"` | if enabled provisioner will create parent datasets for each namespace otherwise, all datasets will be provisioned in a flat manner |
| storageClass.parameters.datasetNamespaceQuota | string | `nil` | Sets a per-namespace quota. example: 5M | 10G | 1T  (M, Mi, MB, MiB, G, Gi, GB, GiB, T, Ti, TB, or TiB). datasetEnableNamespaces must also be set to `true` |
| storageClass.parameters.datasetNamespaceReservation | string | `nil` | Sets a per-namespace reservation. Should not be greater than the `datasetNamespaceQuota`. datasetEnableNamespaces must also be set to `true`. example: 5M | 10G | 1T  (M, Mi, MB, MiB, G, Gi, GB, GiB, T, Ti, TB, or TiB). |
| storageClass.parameters.datasetDeterministicNames | string | `"true"` | if enabled created datasets will adhere to reliable pattern. if datasetNamespaces == true dataset pattern is: <datasetParentName>/<namespace>/<PVC Name>. if datasetNamespaces == false dataset pattern is: <datasetParentName>/<namespace>-<PVC Name>. if disabled, datasets will be created with a name pvc-<uid> (the name of the provisioned PV). |
| storageClass.parameters.datasetRetainPreExisting | string | `"true"` | if enabled and datasetDeterministicNames is enabled then dataset that already exist (pre-provisioned out of band) will be retained by the provisioner during deletion of the reclaim process ignored if datasetDeterministicNames is disabled (collisions result in failure) |
| storageClass.parameters.datasetPermissionsGroup | string | `"wheel"` | Sets group of the dataset mount directory (on FreeNAS) immediately upon creation |
| storageClass.parameters.datasetPermissionsMode | string | `"0777"` | Sets chmod of the dataset mount directory (on FreeNAS) immediately upon creation |
| storageClass.parameters.datasetPermissionsUser | string | `"root"` | Sets user of the dataset mount directory (on FreeNAS) immediately upon creation |
| storageClass.parameters.shareHost | string | `nil` | Determines what the 'server' property of the NFS share will be in kubernetes. Its purpose is to provide flexibility between the control and data planes of FreeNAS. If not set, uses the 'host' value from the secret. |
| storageClass.parameters.shareAlldirs | string | `"true"` | Determines if newly created NFS shares will have the 'All Directories' option checked - note that some k8s versions (e.g OKD 3.11 which has v1.11.0 under the hood) may demand Strings as in "true" or "false". |
| storageClass.parameters.shareAllowedHosts | string | `""` | Authorized hosts (space-separated) allowed to access the shares. All by default. |
| storageClass.parameters.shareAllowedNetworks | string | `""` | Authorized hosts/networks (space-separated) allowed to access the shares. All by default. |
| storageClass.parameters.shareMaprootGroup | string | `"wheel"` | Determines root group mapping. NOTE: cannot be used simultaneously with shareMapAll{User,Group} |
| storageClass.parameters.shareMaprootUser | string | `"root"` | Determines root user mapping. NOTE: cannot be used simultaneously with shareMapAll{User,Group} |
| storageClass.parameters.shareMapallUser | string | `nil` | Determines user mapping for all access (not recommended). NOTE: cannot be used simultaneously with shareMaproot{User,Group} |
| storageClass.parameters.shareMapallGroup | string | `nil` | Determines group mapping for all access (not recommended. NOTE: cannot be used simultaneously with shareMaproot{User,Group} |
| storageClass.parameters.shareRetainPreExisting | string | `"true"` | if enabled and datasetDeterministicNames is enabled then shares that already exist (pre-provisioned out of band) will be retained by the provisioner during deletion of the reclaim process ignored if datasetDeterministicNames is disabled (collisions result in failure) |
| tolerations | list | `[]` | Node toleration configuration |


## Install via Manifests

Run it on the cluster:

```
kubectl apply -f deploy/rbac.yaml -f deploy/deployment.yaml
```

Alternatively, for advanced use-cases you may run the provisioner out of cluster
including directly on the FreeNAS server if desired.  Running out of cluster is
not currently recommended.

```
./bin/freenas-provisioner-freebsd --kubeconfig=/path/to/kubeconfig.yaml
```

### Create `StorageClass` and `Secret`

All the necessary resources are available in the `deploy` folder.  At a minimum
`secret.yaml` must be modified (remember to `base64` the values) to reflect the
server details.  You may also want to read `class.yaml` to review available
`parameters` of the storage class.  For instance to set the `datasetParentName`.

```
kubectl apply -f deploy/secret.yaml -f deploy/class.yaml
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

## Release

- Update the Makefile with the future new version to be released and pushed (docker image)
- Update `deploy/deployment.yaml` with the new image version as well
- Commit, push
- Create a tag:

```
git tag v<version>
git push --tags
```

- Once release is done by Travis, push the new docker image:

```
make push
```

## Docs

 * https://github.com/kubernetes/community/tree/master/contributors/design-proposals/storage
 * https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/volume-provisioning.md
 * https://kubernetes.io/docs/concepts/storage/storage-classes/
 * https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#-strong-api-overview-strong-
 * https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/container-storage-interface.md
 * https://github.com/kubernetes-csi/drivers/tree/master/pkg

## TODO

 * volume resizing - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/grow-volume-size.md
 * volume snapshots - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/volume-snapshotting.md
 * ~~mount options - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/mount-options.md~~
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
