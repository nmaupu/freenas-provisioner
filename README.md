[![Build Status](https://travis-ci.org/nmaupu/freenas-provisioner.svg?branch=master)](https://travis-ci.org/nmaupu/freenas-provisioner)
[![Go Report Card](https://goreportcard.com/badge/github.com/nmaupu/freenas-provisioner)](https://goreportcard.com/report/github.com/nmaupu/freenas-provisioner)

# What is freenas-provisioner

Freenas-provisioner is a Kubernetes external provisioner.
When a `PersisitentVolumeClaim` appears on a Kube cluster, the provisioner will make the corresponding calls to the configured Freenas API to create a dataset and a NFS share usable by the claim. When the claim or the persistent volume is deleted, the provisioner deletes the previously created dataset and share.

See this for more info on external provisioner :
https://github.com/kubernetes-incubator/external-storage

# Building

```
make vendor && make
```
Binary is located into `bin/freenas-provisioner`
It is compiled to be run on `linux-amd64` by default.

# Usage

## Provision the provisioner

Use the following docker image : `docker.io/nmaupu/freenas-provisioner` and create a pod with it.
```
kind: Pod
apiVersion: v1
metadata:
  name: freenas-provisioner
  namespace: kube-system
spec:
  containers:
    - name: freenas-provisioner
      image: docker.io/nmaupu/freenas-provisioner:0.7
      imagePullPolicy: Always
      env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: FREENAS_HOST
          value: 192.168.12.8
        - name: FREENAS_PORT
          value: 443
        - name: FREENAS_USER
          value: root
        - name: FREENAS_PASSWORD
          value: mypass
```

Use a better secret handling for a real usage. Here is just a small quick test.

Run it on the cluster :
```
kubectl apply -f pod.yaml
```

Note : if you cluster run with RBAC support, you need to create according roles for the provisioner to be able to work.
You can use the provided `rbac.yaml` file to do this :
```
kubectl apply -f rbac.yaml
```

## Example usage
Then we can use our new provisioner. First create a *storage class* resource which tells what provisioner to use:
```
---
kind: StorageClass
apiVersion: storage.k8s.io/v1beta1
metadata:
  name: freenas-test-sc
provisioner: maupu.org/freenas
```

Next, create a *persistent volume claim* using that storage class :
```
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: freenas-test-pvc
spec:
  storageClassName: freenas-test-sc
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Mi
```

Use that claim on a testing pod :
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

Create everything :
```
kubectl apply -f usage.yaml
```

The underlying dataset / NFS share should quickly be poping up on Freenas side.
In case of issue, follow the provisioner's logs using :
```
kubectl -n kube-system logs -f freenas-provisioner
```
