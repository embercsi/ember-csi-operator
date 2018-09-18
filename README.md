### Project Status: pre-alpha
The project is currently pre-alpha and it is expected that breaking changes to the API will be made in the upcoming releases.

# ember-csi-operator
Operator to create/configure/manage Ember CSI Driver atop Kubernetes/OpenShift

## Quick Start
$ oc create -f deploy/install.yaml

## Create required secrets which the EmberCSI resource will use
oc create secret generic backend-secret --from-literal=X_CSI_BACKEND_CONFIG='{"volume_backend_name": "rbd", "volume_driver": "cinder.volume.drivers.rbd.RBDDriver", "rbd_user": "cinder", "rbd_pool": "cinder_volumes", "rbd_ceph_conf": "/etc/ceph/ceph.conf", "rbd_keyring_conf": "/etc/ceph/ceph.client.cinder.keyring"}'

oc create secret generic sysfiles-secret --from-file=sysfiles.tar

## Deploy the EmberCSI custom resource
$ kubectl create -f deploy/cr.yaml

## Verify that the pods are created
$ kubectl get pods -n csi
NAME            READY     STATUS    RESTARTS   AGE
ember-csi-operator-786769bdc7-dfl4l   1/1       Running   0          11m
ember-csi-test-controller-0           3/3       Running   0          11m
ember-csi-test-node-2mf5b             2/2       Running   0          11m

