package embercsi

import (
	"context"
	"strings"

	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	snapv1a1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

        "fmt"
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
		glog.Info("Creating a new StatefulSet %s in %s", ss.Name, ss.Namespace)
		err = r.client.Create(context.TODO(), ss)
		if err != nil {
			glog.Error("Failed to create a new StatefulSet %s in %s", ss.Name, ss.Namespace)
			return err
		}
		glog.Info("Successfully Created a new StatefulSet %s in %s", ss.Name, ss.Namespace)
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
		glog.Info("Creating a new Daemonset %s in %s", ds.Name, ds.Namespace)
		err = r.client.Create(context.TODO(), ds)
		if err != nil {
			glog.Error("Failed to create a new Daemonset %s in %s", ds.Name, ds.Namespace)
			return err
		}
		glog.Info("Successfully Created a new Daemonset %s in %s", ds.Name, ds.Namespace)
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
		glog.Info("Creating a new StorageClass %s in %s", sc.Name, sc.Namespace)
		err = r.client.Create(context.TODO(), sc)
		if err != nil {
			glog.Error("Failed to create a new StorageClass %s in %s", sc.Name, sc.Namespace)
			return err
		}
		glog.Info("Successfully Created a new StorageClass %s in %s", sc.Name, sc.Namespace)
	} else if err != nil {
		glog.Error("failed to get StorageClass", err)
		return err
	}

	return nil
}

// construct EnvVars for the Driver Pod
func generateEnvVars(ecsi *embercsiv1alpha1.EmberCSI, driverMode string) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name: "PYTHONUNBUFFERED",
			Value: "0",
		},{
			Name: "CSI_ENDPOINT",
			Value: "unix:///csi-data/csi.sock",
		},{
			Name: "X_CSI_SPEC_VERSION",
			Value: Conf.Sidecars[Cluster].CSISpecVersion,
		},{
                        Name: "X_CSI_EMBER_CONFIG",
                        Value: fmt.Sprintf("%s.%s%s", "{\"plugin_name\": \"io.ember-csi", ecsi.Name, "\", \"project_id\": \"io.ember-csi\", \"user_id\": \"io.ember-csi\", \"root_helper\": \"sudo\", \"request_multipath\": \"true\" }"),
                },
	}

	if driverMode == "controller" {
		envVars = append(envVars, corev1.EnvVar{
					Name: "KUBE_NODE_NAME",
					ValueFrom:  &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				}, corev1.EnvVar{
					Name: "CSI_MODE",
					Value: "controller",
				},
			)
	} else {
		envVars = append(envVars, corev1.EnvVar{
					Name: "X_CSI_NODE_ID",
					ValueFrom:  &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				}, corev1.EnvVar{
					Name: "CSI_MODE",
					Value: "node",
				},
			)
	}

	if len(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_BACKEND_CONFIG",
                        Value: ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG,
			},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_PERSISTENCE_CONFIG) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_PERSISTENCE_CONFIG",
                        Value: ecsi.Spec.Config.EnvVars.X_CSI_PERSISTENCE_CONFIG,
			},
		)
	} else { // Use CRD as the default persistence
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_PERSISTENCE_CONFIG",
                        Value: fmt.Sprintf("{\"storage\":\"crd\",\"namespace\":%s}", ecsi.Namespace),
			},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_DEBUG_MODE) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_DEBUG_MODE",
                        Value: ecsi.Spec.Config.EnvVars.X_CSI_DEBUG_MODE,
			},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_ABORT_DUPLICATES) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_ABORT_DUPLICATES",
                        Value: ecsi.Spec.Config.EnvVars.X_CSI_ABORT_DUPLICATES,
			},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_DEFAULT_MOUNT_FS) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_DEFAULT_MOUNT_FS",
                        Value: ecsi.Spec.Config.EnvVars.X_CSI_DEFAULT_MOUNT_FS,
			},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_NODE_ID) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_NODE_ID",
                        Value: ecsi.Spec.Config.EnvVars.X_CSI_NODE_ID,
			},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_STORAGE_NW_IP) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_STORAGE_NW_IP",
                        Value: ecsi.Spec.Config.EnvVars.X_CSI_STORAGE_NW_IP,
			},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.CSI_ENDPOINT) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "CSI_ENDPOINT",
                        Value: ecsi.Spec.Config.EnvVars.CSI_ENDPOINT,
			},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.CSI_MODE) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "CSI_MODE",
                        Value: ecsi.Spec.Config.EnvVars.CSI_MODE,
			},
		)
	}
	if len(ecsi.Spec.Config.SysFiles.Name) > 0 {
		envVars = append(envVars, corev1.EnvVar{
                        Name: "X_CSI_SYSTEM_FILES",
                        Value: fmt.Sprintf("/tmp/ember-csi/%s", ecsi.Spec.Config.SysFiles.Key),
			},
		)
	}

	return envVars
}

// labelsForEmberCSI returns the labels for selecting the resources
// belonging to the given EmberCSI CR name.
func labelsForEmberCSI(name string) map[string]string {
	return map[string]string{"app": "embercsi", "embercsi_cr": name}
}

// podList returns a corev1.PodList object
func podList() *corev1.PodList {
	return &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// Construct a VolumeMount based on cluster type, secrets, etc
func generateVolumeMounts(ecsi *embercsiv1alpha1.EmberCSI, csiDriverMode string) []corev1.VolumeMount {
	var bidirectional corev1.MountPropagationMode       = corev1.MountPropagationBidirectional
	var hostToContainer corev1.MountPropagationMode     = corev1.MountPropagationHostToContainer

	vm := []corev1.VolumeMount {
		{
			MountPath: "/csi-data",
			Name: "socket-dir",
			MountPropagation: &bidirectional,
		},{
			MountPath: "/etc/iscsi",
			Name: "iscsi-dir",
			MountPropagation: &bidirectional,
		},{
			MountPath: "/var/lib/iscsi",
			Name: "var-lib-iscsi",
			MountPropagation: &bidirectional,
		},{
			MountPath: "/etc/multipath",
			Name: "multipath-dir",
			MountPropagation: &bidirectional,
		},{
			MountPath: "/etc/multipath.conf",
			Name: "multipath-conf",
			MountPropagation: &hostToContainer,
		},{
			MountPath: "/lib/modules",
			Name: "modules-dir",
			MountPropagation: &hostToContainer,
		},{
			MountPath: "/run/udev",
			Name: "run-dir",
			MountPropagation: &hostToContainer,
		},{
			MountPath: "/dev",
			Name: "dev-dir",
			MountPropagation: &bidirectional,
		},{
			MountPath: "/etc/localtime",
			Name: "localtime",
			MountPropagation: &hostToContainer,
		},
	}

	// Check to see if the volume driver is LVM
	if len(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG) > 0 && strings.Contains(strings.ToLower(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG), "lvmvolume") {
		vm = append(vm, corev1.VolumeMount{
			Name: "etc-lvm",
			MountPath: "/etc/lvm",
			MountPropagation: &bidirectional,
		}, corev1.VolumeMount{
			Name: "var-lock-lvm",
			MountPath: "/var/lock/lvm",
			MountPropagation: &bidirectional,
		},
		)
	}

        // Check to see if the X_CSI_SYSTEM_FILES secret is present in the CR
        if len(ecsi.Spec.Config.SysFiles.Name) > 0  {
		vm = append(vm, corev1.VolumeMount{
			Name: "system-files",
			MountPath: "/tmp/ember-csi",
		},
		)
        }

	if csiDriverMode == "node" {
		// Ember CSI shared lock directory to survive restarts
		vm = append(vm, corev1.VolumeMount{
				Name: "shared-lock-dir",
				MountPath: "/var/lib/ember-csi",
				MountPropagation: &bidirectional,
			},
		)

		// ocp
		if strings.Contains(Cluster, "ocp") || Cluster == "default" {
			vm = append(vm, corev1.VolumeMount{
					Name:      "mountpoint-dir",
					MountPropagation: &bidirectional,
					MountPath: "/var/lib/origin/openshift.local.volumes",
				},corev1.VolumeMount{
					MountPath: "/var/lib/kubelet/device-plugins",
					Name: "kubelet-socket-dir",
					MountPropagation: &bidirectional,
				},
			)
		} else {	// k8s
			vm = append(vm, corev1.VolumeMount{
					Name:      "mountpoint-dir",
					MountPropagation: &bidirectional,
					MountPath: "/var/lib/kubelet",
				},
			)
		}
	}

	return vm
}

func generateVolumes (ecsi *embercsiv1alpha1.EmberCSI, csiDriverMode string) []corev1.Volume {
        var dirOrCreate corev1.HostPathType                 = corev1.HostPathDirectoryOrCreate

	vol := []corev1.Volume {
		{
			Name: "run-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/run/udev",
				},
			},
		},{
			Name: "dev-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/dev",
				},
			},
		},{
			Name: "iscsi-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/iscsi",
				},
			},
		},{
			Name: "var-lib-iscsi",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/iscsi",
				},
			},
		},{
			Name: "multipath-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/multipath",
				},
			},
		},{
			Name: "multipath-conf",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/multipath.conf",
				},
			},
		},{
			Name: "modules-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/lib/modules",
				},
			},
		},{
			Name: "localtime",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/localtime",
				},
			},
		},
	}

	// Check to see if the volume driver is LVM
	if len(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG) > 0 && strings.Contains(strings.ToLower(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG), "lvmvolume")  {
		vol = append(vol, corev1.Volume{
			Name: "etc-lvm",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/lvm",
				},
			},
		}, corev1.Volume{
			Name: "var-lock-lvm",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lock/lvm",
				},
			},
		},
		)
	}

	// Check to see if the X_CSI_SYSTEM_FILES secret is present in the CR
	if len(ecsi.Spec.Config.SysFiles.Name) > 0 {
		vol = append(vol, corev1.Volume{
			Name: "system-files",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ecsi.Spec.Config.SysFiles.Name,
					Items: []corev1.KeyToPath{
						{
							Key: ecsi.Spec.Config.SysFiles.Key,
							Path: ecsi.Spec.Config.SysFiles.Key,
						},
					},      
				},
			},
		},
		)
	}

	// The "node" mode of the CSI driver requires mount in /var/lib/kubelet to
	// communicate with the kubelet
	if csiDriverMode == "node" {
		vol = append(vol, corev1.Volume{
				Name: "socket-dir",
				VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: fmt.Sprintf("%s.%s", "/var/lib/kubelet/plugins/io.ember-csi", ecsi.Name),
						},
					},
				},corev1.Volume{
					Name: "shared-lock-dir",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/ember-csi",
							Type: &dirOrCreate,
                                                },
                                        },
                                },
			)
		// ocp
		if strings.Contains(Cluster, "ocp") || Cluster == "default" {
			vol = append(vol, corev1.Volume{
					Name: "mountpoint-dir",
					VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/origin/openshift.local.volumes",
						},
					},
				},corev1.Volume{
					Name: "kubelet-socket-dir",
					VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/kubelet/device-plugins",
							Type: &dirOrCreate,
						},
					},
				},
			)
		} else {	// k8s
			vol = append(vol, corev1.Volume{
					Name: "mountpoint-dir",
					VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/kubelet",
						},
					},
				},
			)
		}
	} else {	// "controller" or "all" mode
		vol = append(vol, corev1.Volume{
				Name: "socket-dir",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	}


	return vol
}
