### Project Status: pre-alpha
The project is currently pre-alpha and it is expected that breaking changes to the API will be made in the upcoming releases.

# ember-csi-operator
Operator to create/configure/manage Ember CSI Driver atop Kubernetes/OpenShift

## Quick Start
$ kubectl create -f deploy/install.yaml

# Create any required configmap which the EmberCSI resource will use
kubectl -n csi create configmap ceph-configs  --from-file=/path/to/ceph.conf --from-file=/path/to/keyring

# Deploy the EmberCSI custom resource
$ kubectl create -f deploy/cr.yaml

# Verify that the pods are created
$ kubectl get pods -n csi
NAME            READY     STATUS    RESTARTS   AGE
ember-csi-operator-786769bdc7-dfl4l   1/1       Running   0          11m
ember-csi-test-controller-0           3/3       Running   0          11m
ember-csi-test-node-2mf5b             2/2       Running   0          11m

