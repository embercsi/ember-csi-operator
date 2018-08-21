### Project Status: pre-alpha
The project is currently pre-alpha and it is expected that breaking changes to the API will be made in the upcoming releases.

# ember-csi-operator
Operator to create/configure/manage Ember CSI Driver atop Kubernetes/OpenShift

## Quick Start

Create and deploy an app-operator using the SDK CLI:

```sh
# Prep Steps
# OpenShift
oc new-project csi
oc adm policy add-scc-to-user privileged -n csi -z csi-controller-sa
oc adm policy add-scc-to-user privileged -n csi -z csi-node-sa

# Kubernetes
kubectl create namespace csi

# Create configmap which the CR will use
kubectl -n csi create configmap ceph-configs  --from-file=/path/to/ceph.conf --from-file=/path/to/keyring

# Deploy the ember-csi-operator
$ kubectl create -f deploy/rbac.yaml
$ kubectl create -f deploy/crd.yaml
$ kubectl create -f deploy/operator.yaml
# The CR will create a statefulset and daemonset 
$ kubectl create -f deploy/cr.yaml

# Verify that the busybox pod is created
$ kubectl get pods -n csi
NAME            READY     STATUS    RESTARTS   AGE
ember-csi-operator-786769bdc7-dfl4l   1/1       Running   0          11m
ember-csi-test-controller-0           3/3       Running   0          11m
ember-csi-test-node-2mf5b             2/2       Running   0          11m

# Cleanup
$ kubectl delete -f deploy/cr.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/crd.yaml
$ kubectl delete -f deploy/rbac.yaml
```

