#!/bin/bash
set -e

CRC=${1:-crc}
SECRET=${2:-pull-secret}
SOURCE=${3:-community-operators}


function do_ssh {
  SSH_PARAMS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~`whoami`/.crc/machines/crc/id_rsa"
  SSH_REMOTE="core@`${CRC} ip`"
  ssh $SSH_PARAMS $SSH_REMOTE "$@"
}

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


# Create LVM VG if it doesn't exist
do_ssh 'sudo bash -c '\''if ! vgdisplay ember-volumes ; then truncate -s 10G /var/lib/containers/ember-volumes && device=`losetup --show -f /var/lib/containers/ember-volumes ` && echo -e \"device is $device\n\" && pvcreate $device && vgcreate ember-volumes $device && vgscan && sed -i "s/^\tudev_sync = 1/\tudev_sync = 0/" /etc/lvm/lvm.conf && sed -i "s/^\tudev_rules = 1/\tudev_rules = 0/" /etc/lvm/lvm.conf; fi'\'

# Ensure iscsid is running on the host, because with CRC sometimes it fails to start on VM start on CRC 1.17
# On CRC 1.20 it fails to start because it doesn't have the initiatorname in the system
do_ssh 'if [[ ! -e /etc/iscsi/initiatorname.iscsi ]]; then echo InitiatorName=`iscsi-iname` | sudo tee /etc/iscsi/initiatorname.iscsi; fi'
do_ssh 'sudo systemctl restart iscsid'

# Multipath doesn't have a configuration, so we need to create it
do_ssh 'if [[ ! -e /etc/multipath.conf ]]; then sudo mpathconf --enable --with_multipathd y --user_friendly_names n --find_multipaths y && sudo systemctl start multipathd; fi'

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
