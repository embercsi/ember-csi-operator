package embercsi

import (
	"context"
	"fmt"
	//"reflect"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	"github.com/golang/glog"
	snapv1a1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new EmberCSI Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

var reconcileEmberCSI *ReconcileEmberCSI

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	reconcileEmberCSI = &ReconcileEmberCSI{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	return reconcileEmberCSI
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("embercsi-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource EmberCSI
	err = c.Watch(&source.Kind{Type: &embercsiv1alpha1.EmberCSI{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch owned objects
	watchOwnedObjects := []runtime.Object{
		&appsv1.StatefulSet{},
		&appsv1.DaemonSet{},
		&storagev1.StorageClass{},
	}
	// Enable objects based on CSI Spec
	if emberCSIOperatorConfig.getCSISpecVersion() >= 1.0 {
		watchOwnedObjects = append(watchOwnedObjects, &snapv1a1.VolumeSnapshotClass{})
	}

	ownerHandler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &embercsiv1alpha1.EmberCSI{},
	}
	for _, watchObject := range watchOwnedObjects {
		err = c.Watch(&source.Kind{Type: watchObject}, ownerHandler)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileEmberCSI{}

// ReconcileEmberCSI reconciles a EmberCSI object
type ReconcileEmberCSI struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	cache  cache.Cache
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a EmberCSI object and makes changes based on the state read
// and what is in the EmberCSI.Spec
func (r *ReconcileEmberCSI) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.V(3).Infof("Reconciling EmberCSI %s/%s\n", request.Namespace, request.Name)

	// Fetch the EmberCSI instance
	instance := &embercsiv1alpha1.EmberCSI{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Warningf("Failed to get %v: %v", request.NamespacedName, err)
		return reconcile.Result{}, err
	}

	// Manage objects created by the operator
	return reconcile.Result{}, r.syncChildren(instance)
}

// Manage the Objects created by the Operator.
func (r *ReconcileEmberCSI) syncChildren(instance *embercsiv1alpha1.EmberCSI) error {
	glog.V(3).Infof("Reconciling EmberCSI Deployment Objects")

	err := r.syncStatefulSet(instance)
	if err != nil {
		return err
	}

	err = r.syncDaemonSet(instance)
	if err != nil {
		return err
	}

	err = r.syncStorageClass(instance)
	if err != nil {
		return err
	}

	err = r.syncVolumeSnapshotClass(instance)
	if err != nil {
		return err
	}

	return nil
}

// Creates a new StatefulSet or updates it based on either spec change or the Operator's config change
func (r *ReconcileEmberCSI) syncStatefulSet(instance *embercsiv1alpha1.EmberCSI) error {
	existing := &appsv1.StatefulSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-controller", instance.Name), Namespace: instance.Namespace}, existing)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create a new object
			actual := r.statefulSetForEmberCSI(instance)
			glog.V(3).Infof("exist: driver image is %s", emberCSIOperatorConfig.getDriverImage(getBackendName(instance)))
			err = r.client.Create(context.TODO(), actual)
			if err != nil {
				if errors.IsAlreadyExists(err) {
					return nil	// Don't requeue. Object exists
				}
				return err
			}
			glog.V(3).Infof("Successfully Created a new Statefulset %s in %s", actual.Name, actual.Namespace)
			return nil
		}
		return err
	}

	// Create a new storage class with latest config changes
	required := r.statefulSetForEmberCSI(instance)
	if isSpecEqual(existing.Spec.Template, required.Spec.Template) {
		glog.V(3).Infof("INFO: Statefulset is in sync. No update needed")
		return nil
	}

	existingCopy := existing // Shallow copy
	existingCopy.Spec = *required.Spec.DeepCopy()

	err = r.client.Update(context.TODO(), existingCopy)
	glog.V(3).Infof("INFO: Statefulset %s updated in %s: %s", existingCopy.Name, existingCopy.Namespace, err)
	return err
}

// Creates a new DaemonSet or updates it based on either spec change or the Operator's config change
func (r *ReconcileEmberCSI) syncDaemonSet(instance *embercsiv1alpha1.EmberCSI) error {
	existing := &appsv1.DaemonSet{}
	daemonSetCount := 1

	// Check whether topology is enabled. We add +1 because
	// of the default daemonset in addition to the topology ones
	if len(instance.Spec.Topologies) > 0 {
		daemonSetCount = len(instance.Spec.Topologies) + 1
	}

	var errs []error
	var err error
	for i := 0; i < daemonSetCount; i++ {
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-node-%d", instance.Name, i), Namespace: instance.Namespace}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				glog.V(3).Infof("Creating a Daemonset with index: %d", i)
				existing = r.daemonSetForEmberCSI(instance, i)
				err = r.client.Create(context.TODO(), existing)
				if err != nil {
					glog.Errorf("Failed to create a new Daemonset %s with index %d in %s: %s", existing.Name, i, existing.Namespace, err)
					//return err
				}
				glog.V(3).Infof("Successfully Created a new Daemonset %s in %s", existing.Name, existing.Namespace)
			}
		}
		errs = append(errs, err)
		if err != nil {
			continue
		}

		// Continue to see if we can update the object
		required := r.daemonSetForEmberCSI(instance, i)
		if isSpecEqual(existing.Spec.Template, required.Spec.Template) {
			glog.V(3).Infof("INFO: DaemonSet with Index %d is in sync. No update needed", i)
			continue // Don't do the update
		}

		existingCopy := existing // Shallow copy
		existingCopy.Spec = *required.Spec.DeepCopy()

		err = r.client.Update(context.TODO(), existingCopy)
		glog.V(3).Infof("INFO: DaemonSet %s with index %d updated in %s: %s", existingCopy.Name, i, existingCopy.Namespace, err)
	}

	// Return any non-zero errors to reconciler to requeue request
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

// Creates a new Storageclass or updates it based on either spec change or the Operator's config change
func (r *ReconcileEmberCSI) syncStorageClass(instance *embercsiv1alpha1.EmberCSI) error {
	existing := &storagev1.StorageClass{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-sc", GetPluginDomainName(instance.Name)), Namespace: instance.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			// Define a new StorageClass
			actual := r.storageClassForEmberCSI(instance)
			err = r.client.Create(context.TODO(), actual)
			if err != nil {
				if errors.IsAlreadyExists(err) {
					return nil	// Don't requeue. Object exists
				}
				return err
			}
			glog.V(3).Infof("Successfully Created a new StorageClass %s in %s", actual.Name, actual.Namespace)
			return nil
		}
		return err
	}

	return nil
}

// Creates a new VolumeSnapshotClass or updates it based on either spec change or the Operator's config change
func (r *ReconcileEmberCSI) syncVolumeSnapshotClass(instance *embercsiv1alpha1.EmberCSI) error {
	snapshotEnabled := true
	if len(instance.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG) > 0 && !isSnapshotEnabled(instance.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG) {
		snapshotEnabled = false
	}

	// Create the VolumeSnapshotClass only if CSI Spec is 1.0 or greater
	if emberCSIOperatorConfig.getCSISpecVersion() >= 1.0 && snapshotEnabled {
		existing := &snapv1a1.VolumeSnapshotClass{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-existing", GetPluginDomainName(instance.Name)), Namespace: instance.Namespace}, existing)

		if err != nil {
			if errors.IsNotFound(err) {
				glog.V(3).Info("Info: Creating new VolumeSnapshotClass")
				// Define a new StorageClass
				actual := r.volumeSnapshotClassForEmberCSI(instance)
				err = r.client.Create(context.TODO(), actual)
				if err != nil {
					if errors.IsAlreadyExists(err) {
						return nil	// Don't requeue. Object exists
					}
					return err
				}
    }
    // Remove the VolumeSnapshotClass and Update the controller and nodes 
    if !snapshotEnabled {
            glog.V(3).Info("Info: Request to disable VolumeSnapshotClass")
            existing := &snapv1a1.VolumeSnapshotClass{}
            err := r.client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-existing", GetPluginDomainName(instance.Name)), Namespace: instance.Namespace}, existing)
            if err != nil && errors.IsAlreadyExists(err) {
                    err = r.client.Delete(context.TODO(), existing)
                    if err != nil {
                            glog.Errorf("Failed to remove VolumeSnapshotClass %s in %s: %s", fmt.Sprintf("%s-existing", GetPluginDomainName(instance.Name)), instance.Namespace, err)
                            return err
                    }
            } else if err != nil {
                    glog.Error("Error: Failed to get VolumeSnapshotClass", err)
                    return err
            }
    }

	return nil
}

// Checks the podspec template to see if new changes have appeared
// Currently checks, container image changes
func isSpecEqual(existing, required corev1.PodTemplateSpec) bool {
	eContainers := existing.Spec.Containers
	rContainers := required.Spec.Containers

	//Array sizes are different. Changes must exist.
	if len(eContainers) != len(rContainers) {
		return false
	}

	// Check whether the container images are the same
	for i, container := range eContainers {
		if container.Image != rContainers[i].Image {
			return false
		}
	}

	// Check whether CSI Spec is the same
	for i, container := range eContainers {
		if container.Name == "ember-csi-driver" {
			for j, env := range container.Env {
				if env.Name == "X_CSI_SPEC_VERSION" {
					if env.Value != rContainers[i].Env[j].Value {
						glog.V(3).Infof("Info: CSI Spec has changed. Sync Required.")
						return false
					}
				}
			}
		}
	}
	return true
}
