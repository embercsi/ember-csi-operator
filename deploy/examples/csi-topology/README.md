This directory contains everything needed to deploy Ember-CSI with CSI Spec v1.0 deployment quickly. It assumes that an Kubernetes 1.13+ cluster are already deployed. For Kuberenetes, ensure that appropriate feature gates are enabled. e.g.
```
$ cat /etc/sysconfig/kubelet
KUBELET_EXTRA_ARGS=--cgroup-driver=systemd --feature-gates=BlockVolume=true,CSIBlockVolume=true,VolumeSnapshotDataSource=true,CSINodeInfo=true,CSIDriverRegistry=true,VolumeScheduling=true
```

The deploy.sh script quickly sets up Ember CSI using the Ember CSI Operator and also deploys an ephemeral Ceph cluster which is used as this deployment's backend plugin.

To launch a RBD-backed Ember-CSI deployment, execute the deploy.sh script:
```
sh deploy.sh
```
