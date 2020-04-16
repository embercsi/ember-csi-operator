#!/bin/bash
set -e

CRC=${1:-crc}
SECRET=${2:-pull-secret}
SOURCE=${3:-community-operators}

# Setup CRC env
${CRC} delete -f || true
${CRC} start -p ${SECRET}
eval $(${CRC} oc-env)
$(${CRC} console --credentials | grep -o "oc login -u kubeadmin.*443")

# Deploy LVM container in the background
oc create -f deploy/examples/lvmbackend.yaml

# Setup custom marketplace to install devel branch of ember operator
oc create -f deploy/examples/catalog.yaml

# Subscribe (install) the operator
oc create -f deploy/examples/operatorgroup.yaml
cat deploy/examples/subscription.yaml | sed -e "s/community-operators/${SOURCE}/g" | oc create -f -

while true; do
  oc wait --for=condition=Ready --timeout=300s -l name=ember-csi-operator pod 2>/dev/null && break
  sleep 5
done

# Deploy LVM backend, PVC, and demo app
oc create \
-f deploy/examples/lvmdriver.yaml \
-f deploy/examples/pvc.yaml \
-f deploy/examples/app.yaml

# Most simple check if volume has been mounted
oc wait --timeout=300s --for=condition=Ready pod my-csi-app
oc exec -it my-csi-app -- df -h /data
