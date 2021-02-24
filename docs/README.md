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

## Catalog build
1. Create a new CSV file using the latest ember-csi container. Replace 0.9.x
   with the next version you want to use.
```
$ pushd build/olm-catalog/
$ CONSOLE_VERSION=4.6 ./olm-csv-gen.sh
$ popd
$ mkdir -p deploy/olm-catalog/0.9.x
$ cp deploy/olm-catalog/0.9.4/ember-csi-operator.crd.yaml deploy/olm-catalog/0.9.x/ember-csi-operator.crd.yaml
$ cp deploy/olm-catalog/next/ember-csi-operator.vX.Y.Z.clusterserviceversion.yaml deploy/olm-catalog/0.9.x/ember-csi-operator.v0.9.x.clusterserviceversion.yaml
```

2. Update deploy/olm-catalog/ember-csi-operator.package.yaml to your next
   version.

3. Build and push a new catalog container

```
$ podman build -f build/Dockerfile.catalog -t quay.io/embercsi/embercsi-catalog:latest deploy/olm-catalog
$ podman push quay.io/embercsi/embercsi-catalog:latest
```

4. Deploy using ember-catalog operator
