package embercsi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	"github.com/golang/glog"
	snapv1b1 "github.com/kubernetes-csi/external-snapshotter/client/v3/apis/volumesnapshot/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new EmberStorageBackend Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEmberStorageBackend{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("embercsi-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource EmberStorageBackend
	err = c.Watch(&source.Kind{Type: &embercsiv1alpha1.EmberStorageBackend{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

        // Watch owned objects
        watchOwnedObjects := []client.Object{
		&corev1.Secret{},
                &appsv1.StatefulSet{},
                &appsv1.DaemonSet{},
                &storagev1.StorageClass{},
                &storagev1.CSIDriver{},
        }
        // Enable objects based on CSI Spec
        if Conf.getCSISpecVersion() >= 1.0 {
               vsc := &snapv1b1.VolumeSnapshotClass{}
               c := mgr.GetClient()
               err := c.Get(context.TODO(), types.NamespacedName{}, vsc)
               if err == nil {
                       watchOwnedObjects = append(watchOwnedObjects, vsc)
               } else {
                       glog.Errorf("%s", err)
               }
        }

        ownerHandler := &handler.EnqueueRequestForOwner{
                IsController: true,
                OwnerType:    &embercsiv1alpha1.EmberStorageBackend{},
        }
        for _, watchObject := range watchOwnedObjects {
                err = c.Watch(&source.Kind{Type: watchObject}, ownerHandler)
                if err != nil {
                        return err
                }
        }



	return nil
}

var _ reconcile.Reconciler = &ReconcileEmberStorageBackend{}

// ReconcileEmberStorageBackend reconciles a EmberStorageBackend object
type ReconcileEmberStorageBackend struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a EmberStorageBackend object and makes changes based on the state read
// and what is in the EmberStorageBackend.Spec
func (r *ReconcileEmberStorageBackend) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Fetch the EmberStorageBackend instance
	instance := &embercsiv1alpha1.EmberStorageBackend{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Warningf("Failed to get %v: %v", request.NamespacedName, err)
		return reconcile.Result{}, err
	}

	glog.V(3).Infof("Reconciling EmberStorageBackend %s in namespace %s\n", request.Name, request.Namespace)

	if len(instance.Spec.Config.SysFiles.Name) > 0 {
		key := types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.Config.SysFiles.Name}
		secret := &corev1.Secret{}
		err := r.client.Get(ctx, key, secret)
		if err != nil {
			glog.Warningf("Failed to get secret %s: %s\n", instance.Spec.Config.SysFiles.Name, err)
		}

		// If multiple keys are found only last one is used
		for key, _ := range secret.Data {
			instance.Spec.Config.SysFiles.Key = key
			glog.Warningf("Found more than one value in Data in secret %s\n", instance.Spec.Config.SysFiles.Name)
		}
		r.client.Update(ctx, instance)
	}
	// Manage objects created by the operator
	return reconcile.Result{}, r.handleEmberStorageBackendDeployment(instance)
}

// Manage the Objects created by the Operator.
func (r *ReconcileEmberStorageBackend) handleEmberStorageBackendDeployment(instance *embercsiv1alpha1.EmberStorageBackend) error {
	persistence_config_json, err := interfaceToString(instance.Spec.Config.EnvVars.X_CSI_PERSISTENCE_CONFIG)
	if err == nil {
		setJsonKeyIfEmpty(&persistence_config_json, "storage", "crd")
	}
	persistence_config_map := make(map[string]interface{})
	_ = json.Unmarshal([]byte(persistence_config_json), &persistence_config_map)
	instance.Spec.Config.EnvVars.X_CSI_PERSISTENCE_CONFIG = persistence_config_map

	backend_config_json, err := interfaceToString(instance.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG)
	if err != nil {
		glog.Errorf("Error parsing X_CSI_BACKEND_CONFIG: %v\n", err)
	}
	setJsonKeyIfEmpty(&backend_config_json, "name", instance.Name)

	// Set multipath option correctly
	backend_cfg, _ := instance.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG.(map[string]interface{})
	multipath, ok := backend_cfg["multipath"]
	if ok {
		ember_cfg, _ := instance.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG.(map[string]interface{})
		ember_cfg["request_multipath"] = multipath
		instance.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG = ember_cfg
	}

	// Redact backend config if needed
	keyName := fmt.Sprintf("ember-csi-operator-%s", instance.Name)
	already_redacted := false
	backend_config_map := make(map[string]interface{})
	err = json.Unmarshal([]byte(backend_config_json), &backend_config_map)
	if err == nil {
		for k, _ := range backend_config_map {
			if backend_config_map[k] == "REDACTED" {
				already_redacted = true
			}
			backend_config_map[k] = "REDACTED"
		}
	}

	// Create or update backend config secret
	key := types.NamespacedName{Namespace: instance.Namespace, Name: keyName}
	secret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), key, secret)
	if err != nil && errors.IsNotFound(err) && already_redacted {
		glog.Errorf("Only redacted X_CSI_BACKEND_CONFIG given and secret %s not found", keyName)
	}
	secret_changed := false
	if !already_redacted {
		secret_data := map[string][]byte {
			"X_CSI_BACKEND_CONFIG": []byte(backend_config_json),
		}
		if err != nil && errors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: keyName,
					Namespace: instance.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:	instance.APIVersion,
							Kind:		instance.Kind,
							Name:		instance.Name,
							UID:		instance.UID,
						},
					},
				},
				Data: secret_data,
				Type: "ember-csi.io/backend-config",
		        }
			err = r.client.Create(context.TODO(), secret)
			if err == nil  {
				glog.V(3).Infof("Created Secret %s", keyName)
				secret_changed = true
			} else {
				glog.Errorf("Failed to create Secret %s: %s", secret.Name, err)
			}
		} else {
			secret.Data = secret_data
			err = r.client.Update(context.TODO(), secret)
			if err == nil  {
				glog.V(3).Infof("Updated Secret %s", keyName)
				secret_changed = true
			} else {
				glog.Errorf("Failed to update Secret %s: %s", secret.Name, err)
			}
		}
	} else {
		glog.V(3).Infof("Secret %s already exists", keyName)
	}

	// Redact credentials, but only if not redacted yet and the
	// secret has been successfully created or updated before
	if !already_redacted && secret_changed {
		instance.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG = backend_config_map
		err = r.client.Update(context.TODO(), instance)
		if err == nil {
			glog.Errorf("Updated EmberStorageBackend %s", instance.Name)
		} else {
			glog.Errorf("EmberStorageBackend instance %s update failed: %s", instance.Name, err)
		}
	}

	// Check if the statefuleSet already exists, if not create a new one
	ss := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-controller", instance.Name), Namespace: instance.Namespace}, ss)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		ss = r.statefulSetForEmberStorageBackend(instance)
		err = r.client.Create(context.TODO(), ss)
		if err != nil {
			glog.Errorf("Failed to create StatefulSet %s: %s", ss.Name, err)
			return err
		} else {
			glog.V(3).Infof("Created StatefulSet %s", ss.Name)
		}
	} else if err != nil {
		glog.Errorf("Failed to get StatefulSet %s: %s", ss.Name, err)
		return err
	} else {
		glog.V(3).Infof("StatefulSet %s already exists", ss.Name)
	}

	// Check if the daemonSet already exists, if not create a new one
	ds := &appsv1.DaemonSet{}
	var dSNotFound []int
	daemonSetIndex := 1

	// Check whether topology is enabled. We add +1 because
	// of the default daemonset in addition to the topology ones
	if len(instance.Spec.Topologies) > 0 {
		daemonSetIndex = len(instance.Spec.Topologies) + 1
	}

	for i := 0; i < daemonSetIndex; i++ {
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-node-%d", instance.Name, i), Namespace: instance.Namespace}, ds)
		if err != nil && errors.IsNotFound(err) {
			dSNotFound = append(dSNotFound, i)
		}
	}
	if len(dSNotFound) > 0 {
		// Define new DaemonSet(s)
		for _, daemonSetIndex := range dSNotFound {
			ds = r.daemonSetForEmberStorageBackend(instance, daemonSetIndex)
			err = r.client.Create(context.TODO(), ds)
			if err != nil {
				glog.Errorf("Failed to create Daemonset %s: %s", ds.Name, err)
				return err
			} else {
				glog.V(3).Infof("Created Daemonset %s", ds.Name)
			}
		}
	} else if err != nil {
		glog.Errorf("Failed to get Daemonset %s: %s", ds.Name, err)
		return err
	} else {
		glog.V(3).Infof("Daemonset %s already exists", ds.Name)
	}

	// Check if the storageclass already exists, if not create a new one
	sc := &storagev1.StorageClass{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: GetPluginDomainName(instance.Name), Namespace: sc.Namespace}, sc)
	if err != nil && errors.IsNotFound(err) {
		// Define a new StorageClass
		sc = r.storageClassForEmberStorageBackend(instance)
		err = r.client.Create(context.TODO(), sc)
		if err != nil {
			glog.Errorf("Failed to create StorageClass %s: %s", sc.Name, err)
			return err
		} else {
			glog.V(3).Infof("Created StorageClass %s", sc.Name)
		}
	} else if err != nil {
		glog.Errorf("Failed to get StorageClass %s: %s", sc.Name, err)
		return err
	} else {
		glog.V(3).Infof("StorageClass %s already exists", sc.Name)
	}

	snapShotEnabled := true
	X_CSI_EMBER_CONFIG, err := interfaceToString(instance.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG)
	if err == nil {
		if !isFeatureEnabled(X_CSI_EMBER_CONFIG, "snapshot") {
			snapShotEnabled = false
		}
	} else {
		glog.Errorf("Error parsing X_CSI_EMBER_CONFIG: %v\n", err)
	}

	// Check if the volumeSnapshotClass already exists, if not create a new one. Only valid with CSI Spec > 1.0
	if Conf.getCSISpecVersion() >= 1.0 && snapShotEnabled {
		vsc := &snapv1b1.VolumeSnapshotClass{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: GetPluginDomainName(instance.Name), Namespace: vsc.Namespace}, vsc)
		if err != nil && errors.IsNotFound(err) {
			// Define a new VolumeSnapshotClass
			vsc = r.volumeSnapshotClassForEmberStorageBackend(instance)
			err = r.client.Create(context.TODO(), vsc)
			if err != nil {
				glog.Errorf("Failed to create VolumeSnapshotClass %s: %s", GetPluginDomainName(instance.Name), err)
				return err
			} else {
				glog.V(3).Infof("Created VolumeSnapshotClass %s", GetPluginDomainName(instance.Name))
			}
		} else if err != nil {
			glog.Errorf("Failed to get VolumeSnapshotClass %s: %s", GetPluginDomainName(instance.Name), err)
		}
	}

	// Remove the VolumeSnapshotClass and Update the controller and nodes 
	if !snapShotEnabled {
		glog.V(3).Info("Info: Request to disable VolumeSnapshotClass")
		vsc := &snapv1b1.VolumeSnapshotClass{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: GetPluginDomainName(instance.Name), Namespace: vsc.Namespace}, vsc)
		if err != nil && !errors.IsNotFound(err) {
			err = r.client.Delete(context.TODO(), vsc)
			if err != nil {
				glog.Errorf("Failed to remove VolumeSnapshotClass %s: %s", GetPluginDomainName(instance.Name), err)
				return err
			} else {
				glog.V(3).Infof("Successfully removed VolumeSnapshotClass %s: %s", GetPluginDomainName(instance.Name), err)
			}
		} else if err != nil {
			glog.Errorf("Failed to get VolumeSnapshotClass %s: %s", GetPluginDomainName(instance.Name), err)
                }
	}

	// Only valid for cluster without using a driver registrar, ie k8s >= 1.13 / ocp >= 4.
	if len(Conf.Sidecars[Cluster].ClusterRegistrar) == 0 && len(Conf.Sidecars[Cluster].Registrar) == 0 {
		// Check if the CSIDriver already exists, if not create a new one
		driver := &storagev1.CSIDriver{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: GetPluginDomainName(instance.Name)}, driver)
		if err != nil && errors.IsNotFound(err) {
			driver = r.csiDriverForEmberStorageBackend(instance)
			err = r.client.Create(context.TODO(), driver)
			if err != nil {
				glog.Errorf("Failed to create CSIDriver %s: %s", driver.Name, err)
				return err
			} else {
				glog.V(3).Infof("Created CSIDriver %s", driver.Name)
			}
		} else if err != nil {
			glog.Errorf("Failed to get CSIDriver %s: %s", driver.Name, err)
			return err
		} else {
			glog.V(3).Infof("CSIDriver %s already exists", driver.Name)
		}
        }

	// Sleep a few seconds, otherwise reconciliation is
	// triggered a couple of times due to new objects
	time.Sleep(time.Second*5)

	return nil
}
