package embercsi

import (
	"fmt"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	snapv1b1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// volumeSnapshotClassForEmberCSI returns a EmberCSI VolumeSnapshotClass object
func (r *ReconcileEmberCSI) volumeSnapshotClassForEmberCSI(ecsi *embercsiv1alpha1.EmberCSI) *snapv1b1.VolumeSnapshotClass {
	ls := labelsForEmberCSI(ecsi.Name)

	vsc := &snapv1b1.VolumeSnapshotClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "snapshot.storage.k8s.io/v1beta1",
			Kind:       "VolumeSnapshotClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vsc", GetPluginDomainName(ecsi.Name)),
			Namespace: ecsi.Namespace,
			Labels:    ls,
		},
		Driver: "ember-csi.io",
		DeletionPolicy: "Delete",
	}

	controllerutil.SetControllerReference(ecsi, vsc, r.scheme)
	return vsc
}
