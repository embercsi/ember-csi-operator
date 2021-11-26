package apis

import (
	"github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1alpha1.SchemeBuilder.AddToScheme,
		snapv1.SchemeBuilder.AddToScheme,
	)
}
