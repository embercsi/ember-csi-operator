package embercsi

import (
	"context"

	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_embercsi")

// Add creates a new EmberCSI Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEmberCSI{client: mgr.GetClient(), scheme: mgr.GetScheme()}
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

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner EmberCSI
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &embercsiv1alpha1.EmberCSI{},
	})
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
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileEmberCSI) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling EmberCSI")

	// Fetch the EmberCSI instance
	instance := &embercsiv1alpha1.EmberCSI{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "failed to get EmberCSI instance")
		return reconcile.Result{}, err
	}

	// Check if the statefuleSet already exists, if not create a new one
	ss := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, ss)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		ss := r.statefulSetForEmberCSI(instance)
		reqLogger.Info("Creating a new StatefulSet", "StatefulSet.Namespace", ss.Namespace, "StatefulSet.Name", ss.Name)
		err = r.client.Create(context.TODO(), ss)
		if err != nil {
			reqLogger.Error(err, "Failed to create a new StatefulSet", "StatefulSet.Namespace", ss.Namespace, "StatefulSet.Name", ss.Name)
			return reconcile.Result{}, err
		}
		// StatefulSet created successfully - return and requeue
		//return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get StatefulSet")
		return reconcile.Result{}, err
	}

	// Check if the daemonSet already exists, if not create a new one
	ds := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, ds)
	if err != nil && errors.IsNotFound(err) {
		// Define a new DaemonSet
		ds := r.daemonSetForEmberCSI(instance)
		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
		err = r.client.Create(context.TODO(), ds)
		if err != nil {
			reqLogger.Error(err, "Failed to create a new DaemonSet", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
			return reconcile.Result{}, err
		}
		// DaemonSet created successfully - return and requeue
		//return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get DaemonSet")
		return reconcile.Result{}, err
	}

	// Check if the storageclass already exists, if not create a new one
	sc := &storagev1.StorageClass{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sc)
	if err != nil && errors.IsNotFound(err) {
		// Define a new StorageClass
		sc := r.storageClassForEmberCSI(instance)
		reqLogger.Info("Creating a new StorageClass", "StorageClass.Namespace", sc.Namespace, "StorageClass.Name", sc.Name)
		err = r.client.Create(context.TODO(), sc)
		if err != nil {
			reqLogger.Error(err, "Failed to create a new StorageClass", "StorageClass.Namespace", sc.Namespace, "StorageClass.Name", sc.Name)
			return reconcile.Result{}, err
		}
		// StorageClass created successfully - return and requeue
		//return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get StorageClass")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

