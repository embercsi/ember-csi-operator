### Debugging
#### Volume mounts fail due to missing directory

If you see an error like this, it's very likely that you need to set your `X_EMBER_OPERATOR_CLUSTER` to your cluster version in your deployment yaml:

```console
# oc describe -n demoapp pod my-csi-app
[...]
Events:
  Type     Reason                  Age                   From                     Message
  ----     ------                  ----                  ----                     -------
  Warning  FailedScheduling        5m2s (x4 over 5m14s)  default-scheduler        pod has unbound immediate PersistentVolumeClaims
  Normal   Scheduled               4m59s                 default-scheduler        Successfully assigned demoapp/my-csi-app to k8s114
  Normal   SuccessfulAttachVolume  4m58s                 attachdetach-controller  AttachVolume.Attach succeeded for volume "pvc-397e465a-8c28-11e9-bf37-525400654e82"
  Warning  FailedMount             41s (x10 over 4m51s)  kubelet, k8s114          MountVolume.MountDevice failed for volume "pvc-397e465a-8c28-11e9-bf37-525400654e82" : rpc error: code = InvalidArgument desc = Parent staging directory for /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-397e465a-8c28-11e9-bf37-525400654e82/globalmount/stage doesn't exist: [Errno 2] No such file or directory: '/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-397e465a-8c28-11e9-bf37-525400654e82/globalmount'
  Warning  FailedMount             38s (x2 over 2m56s)   kubelet, k8s114          Unable to mount volumes for pod "my-csi-app_demoapp(397f2391-8c28-11e9-bf37-525400654e82)": timeout expired waiting for volumes to attach or mount for pod "demoapp"/"my-csi-app". list of unmounted volumes=[my-csi-volume]. list of unattached volumes=[my-csi-volume default-token-jbbmc]
```


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
