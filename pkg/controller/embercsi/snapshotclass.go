package embercsi

import (
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// volumeSnapshotClassForEmberStorageBackend returns a EmberStorageBackend VolumeSnapshotClass object
func (r *ReconcileEmberStorageBackend) volumeSnapshotClassForEmberStorageBackend(ecsi *embercsiv1alpha1.EmberStorageBackend) *snapv1.VolumeSnapshotClass {
	ls := labelsForEmberStorageBackend(ecsi.Name)

	vsc := &snapv1.VolumeSnapshotClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "snapshot.storage.k8s.io/v1",
			Kind:       "VolumeSnapshotClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetPluginDomainName(ecsi.Name),
			Namespace: ecsi.Namespace,
			Labels:    ls,
		},
		Driver: GetPluginDomainName(ecsi.Name),
		DeletionPolicy: "Delete",
	}

	controllerutil.SetControllerReference(ecsi, vsc, r.scheme)
	return vsc
}
