package embercsi

import (
	"fmt"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	snapv1a1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// volumeSnapshotClassForEmberCSI returns a EmberCSI VolumeSnapshotClass object
func (r *ReconcileEmberCSI) volumeSnapshotClassForEmberCSI(ecsi *embercsiv1alpha1.EmberCSI) *snapv1a1.VolumeSnapshotClass {
	ls := labelsForEmberCSI(ecsi.Name)

	return &snapv1a1.VolumeSnapshotClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "snapshot.storage.k8s.io/v1alpha1",
			Kind:       "VolumeSnapshotClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vsc", GetPluginDomainName(ecsi.Name)),
			Namespace: ecsi.Namespace,
			Labels:    ls,
		},
		Snapshotter: GetPluginDomainName(ecsi.Name),
	}
}
