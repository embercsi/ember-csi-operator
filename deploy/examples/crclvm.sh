#!/bin/bash
set -e

CRC=${1:-crc}
SECRET=${2:-pull-secret}
SOURCE=${3:-community-operators}

# Setup CRC env
[ ! -e ${SECRET} ] && echo '{"auths":{"fake":{"auth": "bar"}}}' > ${SECRET}
${CRC} delete -f || true
${CRC} start -p ${SECRET}
eval $(${CRC} oc-env)
$(${CRC} console --credentials | grep -o "oc login -u kubeadmin.*443")

# Enable the csi-snapshot-controller-operator. This has been disabled in crc to
# save some memory, details on these commands in:
# https://code-ready.github.io/crc/#starting-monitoring-alerting-telemetry_gsg
ID=`oc get clusterversion version -ojsonpath='{range .spec.overrides[*]}{.name}{"\n"}{end}' | nl -v 0 -w 1 | grep csi-snapshot-controller-operator | cut -f 1`
oc patch clusterversion/version --type='json' -p '[{"op":"remove", "path":"/spec/overrides/'${ID}'"}]'

# Deploy LVM container in the background
oc create -f deploy/examples/lvmbackend.yaml

# Setup custom marketplace to install devel branch of ember operator
if [ "$SOURCE" != "community-operators" ]; then
  oc create -f deploy/examples/catalog.yaml
fi

# Subscribe (install) the operator
oc create -f deploy/examples/operatorgroup.yaml
sed -e "s/community-operators/${SOURCE}/g" deploy/examples/subscription.yaml | oc create -f -

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
