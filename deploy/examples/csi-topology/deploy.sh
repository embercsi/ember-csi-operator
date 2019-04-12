#!/usr/bin/env bash

# Install a Ceph demo container
kubectl create -f 00-ceph-demo.yml

echo "Wait until the pod is ready..."
kubectl wait -n ceph-demo --timeout=300s --for=condition=Ready pod/ceph-demo-pod

# Create Ember CSI namespace, RBAC and CRDs
kubectl -n ember-csi create -f 01-pre.yml

# Create the required secret
[ -e etc ] && rm -rf etc
kubectl -n ceph-demo cp ceph-demo/ceph-demo-pod:/etc/ceph/ etc/ceph/
echo -e "\n[client]\nrbd default features = 3\n" >> etc/ceph/ceph.conf
tar cf system-files.tar etc/ceph/ceph.conf etc/ceph/ceph.client.admin.keyring
kubectl create -n ember-csi secret generic system-files --from-file=system-files.tar
[ -e system-files.tar ] && rm -f system-files.tar
[ -e etc ] && rm -rf etc

# Deploy the operator
kubectl -n ember-csi create -f 02-operator.yml

echo "Wait until the Operator is ready..."
kubectl wait -n ceph-demo --timeout=300s --for=condition=Ready pod/ember-csi-operator

# Instantiate an Ember CSI instance backed by the Ceph Demo deployment
kubectl -n ember-csi create -f 03-rbd.yml

