### Project Status: pre-alpha
The project is currently pre-alpha and it is expected that breaking changes to the API will be made in the upcoming releases.

# ember-csi-operator
Operator to create/configure/manage Ember CSI Driver atop Kubernetes/OpenShift

## Build
To build the operator, clone this repo into your GOPATH and run make. NOTE: Please ensure that the container image repo and tag are customized.
```
$ mkdir -p ${GOPATH}/src/github.com/embercsi
$ git clone -b devel https://github.com/embercsi/ember-csi-operator
$ cd ember-csi-operator
$ make build
```
If the used Docker release supports multistage builds, you can enable this by setting the MULTISTAGE_BUILD env var:
```
$ MULTISTAGE_BUILD=1 make build
```

## Quick Start
The provided deploy/00-pre.yaml, deploy/01-scc.yaml and deploy/02-operator.yaml files will construct all the necessary RBAC, SCC, Service Accounts, Namespace, etc to run the Ember CSI operator. NOTE: Edit the 02-operator.yaml file if you wish to use your container image. By default it uses docker.io/embercsi/ember-csi-operator:latest

An optional environmental variable of X_EMBER_OPERATOR_CLUSTER can be passed to the operator to enable/disable CSI spec versions as well as selecting the appropriate sidecar and driver images. The default value of X_EMBER_OPERATOR_CLUSTER is "default" which enables CSI spec v0.3. 

```
$ make deploy
```

## Create a Custom Resource File
The Custom Resource File is a yaml file that configures the Ember CSI driver. Details such as unique name, driver backend-specific information and files are specified here. 
### Example CR file.
```
apiVersion: "ember-csi.io/v1alpha1"
kind: "EmberCSI"
metadata:
  name: "my-ceph"
spec:
# Optional. Use nodeSelector for placing CSI controller pod
# nodeSelector:
#   # Any arbitrary key: value pair that must match nodeLabels set by the admin
#   # kubectl get nodes --show-labels
#   ember-csi-controller: true
  config:
    envVars:
      X_CSI_PERSISTENCE_CONFIG:       '{"storage":"crd"}'
      X_CSI_BACKEND_CONFIG :          '{"name": "rbd", "driver": "RBD", "rbd_user": "cinder", "rbd_pool": "cinder_volumes", "rbd_ceph_conf": "/etc/ceph/ceph.conf", "rbd_keyring_conf": "/etc/ceph/keyring"}'
    sysFiles:
      name: sysfiles-secret
      key: "system-files.tar"
```

The name entry will ensure a unique deployment of Ember CSI instance. In the config.envvars section, environment variables specified here are passed to the Ember CSI driver pod. The config.sysFiles entry, specifies the name of the secret, which contains any backend-specific files tar'ed and optionally compressed via gzip or bzip2.

### Enviroment Variables

The CSI driver is configured via environmental variables, any value that doesn't have a default is a required value.

| Name                       | Role       | Description                                                   | Default                                                                                                      | Example                                                                                                                                                                                                                                 |
| -------------------------- | ---------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `CSI_ENDPOINT`             | all        | IP and port to bind the service                               | [::]:50051                                                                                                   | 192.168.1.22:50050                                                                                                                                                                                                                      |
| `CSI_MODE`                 | all        | Role the service should perform: controller, node, all        | all                                                                                                          | controller                                                                                                                                                                                                                              |
| `X_CSI_STORAGE_NW_IP`      | node       | IP address in the Node used to connect to the storage         | IP resolved from Node's fqdn                                                                                 | 192.168.1.22                                                                                                                                                                                                                            |
| `X_CSI_NODE_ID`            | node       | ID used by this node to identify itself to the controller     | Node's fqdn                                                                                                  | csi_test_node                                                                                                                                                                                                                           |
| `X_CSI_PERSISTENCE_CONFIG` | all        | Configuration of the `cinderlib` metadata persistence plugin. | {'storage': 'db', 'connection': 'sqlite:///db.sqlite'}                                                       | {'storage': 'db', 'connection': 'mysql+pymysql://root:stackdb@192.168.1.1/cinder?charset=utf8'}                                                                                                                                         |
| `X_CSI_EMBER_CONFIG`       | all        | Global `cinderlib` configuration                              | {'project_id': 'io.ember-csi', 'user_id': 'io.ember-csi', 'root_helper': 'sudo', 'request_multipath': true } | {"project_id":"k8s project","user_id":"csi driver","root_helper":"sudo"}                                                                                                                                                                |
| `X_CSI_BACKEND_CONFIG`     | controller | Driver configuration                                          |                                                                                                              | {"name": "rbd", "driver": "RBD", "rbd_user": "cinder", "rbd_pool": "volumes", "rbd_ceph_conf": "/etc/ceph/ceph.conf", "rbd_keyring_conf": "/etc/ceph/ceph.client.cinder.keyring"} |
| `X_CSI_DEFAULT_MOUNT_FS`   | node       | Default mount filesystem when missing in publish calls        | ext4                                                                                                         | btrfs                                                                                                                                                                                                                                   |
| `X_CSI_SYSTEM_FILES`       | all        | All required storage driver-specific files archived in tar, tar.gz or tar.bz2 format|                                                                                        | /path/to/etc-ceph.tar.gz                                                                                                                                                                                                                |
| `X_CSI_DEBUG_MODE`         | all        | Debug mode (rpdb, pdb) to use. Disabled by default.           |                                                                                                              | rpdb                                                                                                                                                                                                                                    |
| `X_CSI_ABORT_DUPLICATES`   | all        | If we want to abort or queue (default) duplicated requests.   | false                                                                                                        | true                                                                                                                                                                                                                                    |

### Create required secrets which the EmberCSI resource will use

#### Create the tar file

##### Example for ceph backend

- Prepare the following tree structure:
    - etc/ceph/ceph.conf
    - etc/ceph/keyring

- Create the archive:
    ```
    $ tar -cvf system-files.tar etc
    ```
    
Notes:

- The keyring file is renamed for brevity purposes. 
  It depends on the specified client user. E.g. ceph.client.cinder.keyring

- Add 'rbd default features = 3' to ceph.conf file in order to enable only layering.
  Otherwise, volume creation might fail due to unsupported features (fast-diff/deep-flatten) on old kernels. 

#### Deploy the sysfiles-secret
```
oc create secret generic sysfiles-secret --from-file=system-files.tar
```

## Switch to ember-csi project
```
$ oc project ember-csi
```

## Deploy the Custom Resource
```
$ oc create -f deploy/examples/drivers/ceph.yaml
```

## Verify that the pods are created and the Storageclass exists
```
$ oc get pods -n ember-csi
NAME            READY     STATUS    RESTARTS   AGE
ember-csi-operator-786769bdc7-dfl4l   1/1       Running   0          11m
ember-csi-test-controller-0           3/3       Running   0          11m
ember-csi-test-node-2mf5b             2/2       Running   0          11m
$ oc get storageclass -n ember-csi
NAME                            PROVISIONER                  AGE
my-ceph-sc.ember-csi.io   	my-ceph.ember-csi.io   	     5s
```

## Uninstallation
Before uninstalling the operator, make sure all the pods and PVCs using Ember CSI is cleaned up. After these are cleaned up, run make undeploy

```
$ oc delete -f deploy/examples/app.yaml
$ oc delete -f deploy/examples/pvc.yaml
$ oc delete -f deploy/examples/drivers/ceph.yaml
$ make undeploy
```
