package embercsi

import (
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"github.com/golang/glog"
)

// storageClassForEmberCSI returns a EmberCSI StorageClass object
func (r *ReconcileEmberCSI) storageClassForEmberCSI(ecsi *embercsiv1alpha1.EmberCSI) *storagev1.StorageClass {
	ls := labelsForEmberCSI(ecsi.Name)

	// This binding mode is the default
	volumeBindingMode := storagev1.VolumeBindingImmediate

	// Check whether topology is enabled. If yes, set VolumeBindingMode appropriately
	if len(ecsi.Spec.Topologies) > 0 {
		volumeBindingMode = storagev1.VolumeBindingWaitForFirstConsumer
	}

	expandEnabled := true
	X_CSI_EMBER_CONFIG, err := interfaceToString(ecsi.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG)
	if err == nil {
		if !isFeatureEnabled(X_CSI_EMBER_CONFIG, "expand") {
			expandEnabled = false
		}
	} else {
		glog.Errorf("Error parsing X_CSI_EMBER_CONFIG: %v\n", err)
	}

	sc := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "storage.k8s.io/v1",
			Kind:       "StorageClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetPluginDomainName(ecsi.Name),
			Namespace: ecsi.Namespace,
			Labels:    ls,
		},
		Provisioner:       GetPluginDomainName(ecsi.Name),
		VolumeBindingMode: &volumeBindingMode,
		AllowVolumeExpansion: &expandEnabled,
	}

	controllerutil.SetControllerReference(ecsi, sc, r.scheme)
	return sc
}
