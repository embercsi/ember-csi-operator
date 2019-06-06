### OLM Catalog For Ember CSI

## Deployment

```
$ kubectl apply -f 00-pre.yaml 
namespace/ember-csi created
serviceaccount/ember-csi-operator created
customresourcedefinition.apiextensions.k8s.io/embercsis.ember-csi.io configured
role.rbac.authorization.k8s.io/ember-csi-operator created
rolebinding.rbac.authorization.k8s.io/ember-csi-operator created
clusterrole.rbac.authorization.k8s.io/ember-csi-operator created
clusterrolebinding.rbac.authorization.k8s.io/ember-csi-operator created
customresourcedefinition.apiextensions.k8s.io/embercsis.ember-csi.io configured
$ kubectl apply -f 03-operatorgroup.yaml 
operatorgroup.operators.coreos.com/ember-csi-operator-group created
$ kubectl create -f 0.0.1/ember-csi-operator.v0.0.1.clusterserviceversion.yaml 
clusterserviceversion.operators.coreos.com/ember-csi-operator.v0.0.1 created
$
```

After the CSV is created, wait for its `PHASE` to change from `Pending` to `Installing` to `Succeeded`. Once its in `Succeeded` phase, we can inspect to see whether the ember-csi-operator is running correctly.

```
$ kubectl -n ember-csi get csv,all
NAME                                                                   DISPLAY              VERSION   REPLACES   PHASE
clusterserviceversion.operators.coreos.com/ember-csi-operator.v0.0.1   Ember CSI Operator   0.0.1                Succeeded

NAME                                      READY   STATUS    RESTARTS   AGE
pod/ember-csi-operator-68876fd968-gzb2z   1/1     Running   0          43m

NAME                                 READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/ember-csi-operator   1/1     1            1           43m

NAME                                            DESIRED   CURRENT   READY   AGE
replicaset.apps/ember-csi-operator-68876fd968   1         1         1       43m
```

## OKD
The OKD GUI enables end-users to manage Kubernetes objects as well as interaction with Operators via a web portal.

Before running the GUI, ensure that the `kube-system:default` service account has sufficient permissions to manage objects in various namespaces.

We can do that by creating a `clusterrolebinding` and adding the user into an `admin clusterrrole`.
```
kubectl create clusterrolebinding add-on-cluster-admin   --clusterrole=admin   --serviceaccount=kube-system:default
```

After that we can descend into the operator-lifecycle-manager repo and run make. Ensure that `kubectl` is in the path and points to a working cluster. The `jq` binary should also be in the default path.
```
$ git clone https://github.com/operator-framework/operator-lifecycle-manager.git
$ cd operator-lifecycle-manager
$ make run-console-local
/bin/bash: go: command not found
/bin/bash: go: command not found
Running script to run the OLM console locally:
. ./scripts/run_console_local.sh
Trying to pull repository quay.io/openshift/origin-console ... 
latest: Pulling from quay.io/openshift/origin-console
Digest: sha256:f255feaaad9cbbbf8d19c965f86e12e0a80f1fba5f396e6bade03fce482574b3
Status: Image is up to date for quay.io/openshift/origin-console:latest
Using https://192.168.122.181:6443
The OLM is accessible via web console at:
http://localhost:9000/
Press Ctrl-C to quit

```

Use a browser to connect to http://localhost:9000/ and interact with Operators and other objects.
