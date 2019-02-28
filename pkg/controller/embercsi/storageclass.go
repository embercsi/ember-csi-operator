package embercsi

import (
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        storagev1 "k8s.io/api/storage/v1"
        "fmt"
)

// storageClassForEmberCSI returns a EmberCSI StorageClass object
func (r *ReconcileEmberCSI) storageClassForEmberCSI(ecsi *embercsiv1alpha1.EmberCSI) *storagev1.StorageClass {
	ls := labelsForEmberCSI(ecsi.Name)

	return &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "storage.k8s.io/v1",
			Kind:       "StorageClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sc", PluginDomainName),
			Namespace: ecsi.Namespace,
			Labels:	   ls,
		},
		Provisioner: PluginDomainName,
	}
}
