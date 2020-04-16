#!/bin/bash
set -x
sed -i -e 's/use_lvmetad.*/use_lvmetad = 0/g' /etc/lvm/lvm.conf
sed -i -e 's/udev_sync.*/udev_sync = 0/g' /etc/lvm/lvm.conf
sed -i -e 's/udev_rules.*/udev_rules = 0/g' /etc/lvm/lvm.conf

if [[ ! -f /etc/iscsi/initiatorname.iscsi ]]; then
    echo InitiatorName=$(/sbin/iscsi-iname) | tee /etc/iscsi/initiatorname.iscsi
fi

vgscan --cache

if ! vgdisplay | grep -q 'ember-volumes'; then
	truncate -s 10G /mnt/ember-volumes
	LOOPDEV=$(losetup --show -f /mnt/ember-volumes)

	# Pod will be restarted if the previous command failed
	[[ -z "$LOOPDEV" ]] && exit 1

	pvcreate ${LOOPDEV}
	vgcreate ember-volumes ${LOOPDEV}
	vgscan --cache
fi

echo "$1 done."
