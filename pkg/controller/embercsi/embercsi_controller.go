package embercsi

import (
	"context"

	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	snapv1a1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
        appsv1 "k8s.io/api/apps/v1"
        storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"github.com/golang/glog"
)

// Add creates a new EmberCSI Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEmberCSI{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
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
	if CSI_SPEC > 1.0 {
		watchOwnedObjects = append(watchOwnedObjects, &snapv1a1.VolumeSnapshotClass{})
	}

	ownerHandler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &embercsiv1alpha1.EmberCSI{},
	}
	for _, watchObject := range watchOwnedObjects {
		glog.V(3).Infof("Watching Owned Object: %s/%s\n", "", "")
		err = c.Watch(&source.Kind{Type: watchObject}, ownerHandler)
		if err != nil {
			return err
		}
	}
        if err != nil {
                return err
        }

	return nil
}

var _ reconcile.Reconciler = &ReconcileEmberCSI{}

// ReconcileEmberCSI reconciles a EmberCSI object
type ReconcileEmberCSI struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
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
	return reconcile.Result{}, r.handleEmberCSIDeployment(instance)
}

// Manage the Objects created by the Operator. 
func (r *ReconcileEmberCSI) handleEmberCSIDeployment(instance *embercsiv1alpha1.EmberCSI) error {
	// Check if the statefuleSet already exists, if not create a new one
	ss := &appsv1.StatefulSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, ss)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		ss = r.statefulSetForEmberCSI(instance)
		glog.Infof("Creating a new StatefulSet %s in %s", ss.Name, ss.Namespace)
		err = r.client.Create(context.TODO(), ss)
		if err != nil {
			glog.Errorf("Failed to create a new StatefulSet %s in %s: %s", ss.Name, ss.Namespace, err)
			return err
		}
		glog.Infof("Successfully Created a new StatefulSet %s in %s", ss.Name, ss.Namespace)
	} else if err != nil {
		glog.Error("failed to get StatefulSet", err)
		return err
	}

	// Check if the daemonSet already exists, if not create a new one
	ds := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, ds)
	if err != nil && errors.IsNotFound(err) {
		// Define a new DaemonSet
		ds = r.daemonSetForEmberCSI(instance)
		glog.Infof("Creating a new Daemonset %s in %s", ds.Name, ds.Namespace)
		err = r.client.Create(context.TODO(), ds)
		if err != nil {
			glog.Errorf("Failed to create a new Daemonset %s in %s: %s", ds.Name, ds.Namespace, err)
			return err
		}
		glog.Infof("Successfully Created a new Daemonset %s in %s", ds.Name, ds.Namespace)
	} else if err != nil {
                glog.Error("failed to get DaemonSet", err)
		return err
	}

	// Check if the storageclass already exists, if not create a new one
	sc := &storagev1.StorageClass{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sc)
	if err != nil && errors.IsNotFound(err) {
		// Define a new StorageClass
		sc = r.storageClassForEmberCSI(instance)
		glog.Infof("Creating a new StorageClass %s in %s", sc.Name, sc.Namespace)
		err = r.client.Create(context.TODO(), sc)
		if err != nil {
			glog.Errorf("Failed to create a new StorageClass %s in %s: %s", sc.Name, sc.Namespace, err)
			return err
		}
		glog.Infof("Successfully Created a new StorageClass %s in %s", sc.Name, sc.Namespace)
	} else if err != nil {
		glog.Error("failed to get StorageClass", err)
		return err
	}

	return nil
}

