package stub

import (
	"context"

	"github.com/kirankt/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"

	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
//	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
}

// Constants we'll reference throughout

const (
	// Node DaemonSet's ServiceAccount
	NodeSA string 		= "csi-node-sa"
	// Controller StatefulSet's ServiceAccount
	ControllerSA string 	= "csi-controller-sa"

	// Image Versions
	RegistrarVersion string   = "v0.3.0"
	AttacherVersion string 	  = "v0.3.0"
	ProvisionerVersion string = "v0.3.0"
	DriverVersion string 	  = "master"
)

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.EmberCSI:

		ecsi := o
		var err error

		// Ignore the delete event since the garbage collector will clean up all secondary resources for the CR
		// All secondary resources must have the CR set as their OwnerReference for this to be the case
		if event.Deleted {
			return nil
		}

		logrus.Infof("Creating StatefulSet for Ember CSI Controller")
		// Create the Controller StatefulSet if it doesn't exist
		ss := statefulSetForEmberCSI(ecsi)
		err = sdk.Create(ss)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create controller statefulset: %v", err)
		}

		// Ensure the StatefulSet size is the same as the spec
		err = sdk.Get(ss)
		if err != nil {
			return fmt.Errorf("failed to get StatefulSet: %v", err)
		}
		size := ecsi.Spec.Size
		if *ss.Spec.Replicas != size {
			*ss.Spec.Replicas = size
			err = sdk.Update(ss)
			if err != nil {
				return fmt.Errorf("failed to update StatefulSet: %v", err)
			}
		}

		logrus.Infof("Creating DaemonSet for Ember CSI Nodes")
		ds := daemonSetForEmberCSI(ecsi)
		err = sdk.Create(ds)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create node daemonset: %v", err)
		}


	}
	return nil
}

// statefulSetForEmberCSI returns a EmberCSI StatefulSet object
func statefulSetForEmberCSI(ecsi *v1alpha1.EmberCSI) *appsv1.StatefulSet {
	ls := labelsForEmberCSI(ecsi.Name)

	// There *must* only be one replica
	var replicas int32 	= 1

	//secretName := ecsi.Spec.Secret
	backendConfig 		:= ecsi.Spec.Config.BackendConfig
	persistenceConfig 	:= ecsi.Spec.Config.PersistenceConfig
	configMapName		:= ecsi.Spec.ConfigMap
	trueVar 		:= true

	ss := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-controller", ecsi.Name),
			Namespace: ecsi.Namespace,
			Labels:	   ls,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:    "external-attacher",
						Image:   fmt.Sprintf("%s:%s", "quay.io/k8scsi/csi-attacher:v0.3.0", AttacherVersion),
						Args: []string{"--v=5", "--csi-address=/csi-data/csi.sock"},
						SecurityContext: &v1.SecurityContext{
							Privileged: &trueVar,
						},
						VolumeMounts: []v1.VolumeMount{
							{
								MountPath: "/csi-data", 
								Name: "socket-dir",
							},{
								MountPath: "/dev", 
								Name: "dev-dir",
							},{
								MountPath: "/etc/localtime", 
								Name: "localtime",
							},
						},
					},{
						Name:    "external-provisioner",
						Image:   fmt.Sprintf("%s:%s", "quay.io/k8scsi/csi-provisioner:v0.3.0", ProvisionerVersion),
						Args: []string{"--v=5", "--csi-address=/csi-data/csi.sock", "--provisioner=io.ember-csi"},
						SecurityContext: &v1.SecurityContext{
							Privileged: &trueVar,
						},
						VolumeMounts: []v1.VolumeMount{
							{
								MountPath: "/csi-data", 
								Name: "socket-dir",
							},{
								MountPath: "/dev", 
								Name: "dev-dir",
							},{
								MountPath: "/etc/localtime", 
								Name: "localtime",
							},
						},
					},{
						Name:    "ember-csi-driver",
						Image:   fmt.Sprintf("%s:%s", "akrog/ember-csi:master", DriverVersion),
						SecurityContext: &v1.SecurityContext{
							Privileged: &trueVar,
						},
						Env: []v1.EnvVar{
							{
								Name: "PYTHONUNBUFFERED",
								Value: "0",
							},{
								Name: "CSI_ENDPOINT",
								Value: "unix:///csi-data/csi.sock",
							},{
								Name: "CSI_MODE",
								Value: "controller",
							},{
								Name: "X_CSI_PERSISTENCE_CONFIG",
								Value: persistenceConfig,
							},{
								Name: "X_CSI_BACKEND_CONFIG",
								Value: backendConfig,
								/* ValueFrom: &v1.EnvVarSource{
									SecretKeyRef: &v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: secretName},
										Key:  "backend_config",
									},
								},*/
							},
						},
						VolumeMounts: []v1.VolumeMount{
							{
								MountPath: "/csi-data", 
								Name: "socket-dir",
							},{
								MountPath: "/dev", 
								Name: "dev-dir",
							},{
								MountPath: "/etc/localtime", 
								Name: "localtime",
							},{
								MountPath: "/etc/ceph",
								Name: "config",
							},
						},
					}},
					Volumes: []v1.Volume{
						{
							Name: "socket-dir",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
                                                },{
                                                        Name: "dev-dir",
                                                        VolumeSource: v1.VolumeSource{
                                                                HostPath: &v1.HostPathVolumeSource{
                                                                        Path: "/dev",
                                                                },
                                                        },
                                                },{
                                                        Name: "localtime",
                                                        VolumeSource: v1.VolumeSource{
                                                                HostPath: &v1.HostPathVolumeSource{
                                                                        Path: "/etc/localtime",
                                                                },
                                                        },
                                                },{
							Name: "config",
								VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{Name:configMapName},
								},
							},
						},
					},
				},
			},
		},
	}
	addOwnerRefToObject(ss, asOwner(ecsi))
	return ss
}

// labelsForEmberCSI returns the labels for selecting the resources
// belonging to the given EmberCSI CR name.
func labelsForEmberCSI(name string) map[string]string {
	return map[string]string{"app": "embercsi", "embercsi_cr": name}
}

// addOwnerRefToObject appends the desired OwnerReference to the object
func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}

// asOwner returns an OwnerReference set as the EmberCSI CR
func asOwner(ecsi *v1alpha1.EmberCSI) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: ecsi.APIVersion,
		Kind:       ecsi.Kind,
		Name:       ecsi.Name,
		UID:        ecsi.UID,
		Controller: &trueVar,
	}
}

// podList returns a v1.PodList object
func podList() *v1.PodList {
	return &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []v1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// daemonSetForEmberCSI returns a EmberCSI DaemonSet object
func daemonSetForEmberCSI(ecsi *v1alpha1.EmberCSI) *appsv1.DaemonSet {
	ls := labelsForEmberCSI(ecsi.Name)

	var dirOrCreate v1.HostPathType 		= v1.HostPathDirectoryOrCreate
	var bidirectional v1.MountPropagationMode 	= v1.MountPropagationBidirectional
	var hostToContainer v1.MountPropagationMode 	= v1.MountPropagationHostToContainer
	//secretName 		:= ecsi.Spec.Secret
	backendConfig 		:= ecsi.Spec.Config.BackendConfig
	persistenceConfig 	:= ecsi.Spec.Config.PersistenceConfig
	configMapName		:= ecsi.Spec.ConfigMap
	trueVar 		:= true

	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
                        Name:      fmt.Sprintf("%s-node", ecsi.Name),
                        Namespace: ecsi.Namespace,
			Labels:    ls,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
                        Template: v1.PodTemplateSpec{
                                ObjectMeta: metav1.ObjectMeta{
                                        Labels: ls,
                                },
                                Spec: v1.PodSpec{
					ServiceAccountName: NodeSA,
					HostNetwork: true,
                                        Containers: []v1.Container{
						{
							Name:		"driver-registrar",
							Image:		fmt.Sprintf("%s:%s","quay.io/k8scsi/driver-registrar",RegistrarVersion),
							ImagePullPolicy: v1.PullAlways,
							Args: 		 []string{"--v=5", "--csi-address=/csi-data/csi.sock"},
							SecurityContext: &v1.SecurityContext{
								Privileged: &trueVar,
							},
							Env:	[]v1.EnvVar{
								{
									Name: "KUBE_NODE_NAME",
									ValueFrom:  &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									MountPath: "/csi-data", 
									Name: 	   "socket-dir",
								},{
									MountPath: 	  "/etc/localtime", 
									Name: 		  "localtime",
									MountPropagation: &hostToContainer,
								},{
									MountPath: 	  "/var/lib/origin/openshift.local.volumes", 
									Name: 		  "mountpoint-dir",
									MountPropagation: &bidirectional,
								},{
									MountPath: 	  "/var/lib/kubelet/device-plugins", 
									Name: 		  "kubelet-socket-dir",
									MountPropagation: &bidirectional,
								},
							},
						},{
							Name:		"ember-csi-driver",
							Image:		fmt.Sprintf("%s:%s", "akrog/ember-csi:master", DriverVersion),
							ImagePullPolicy: v1.PullAlways,
							SecurityContext: &v1.SecurityContext{
								Privileged: 		  &trueVar,
								AllowPrivilegeEscalation: &trueVar,
							},
							TerminationMessagePath: "/tmp/termination-log",
							Env:	[]v1.EnvVar{
								{
									Name: "PYTHONUNBUFFERED",
									Value: "0",
								},{
									Name: "CSI_ENDPOINT",
									Value: "unix:///csi-data/csi.sock",
								},{
									Name: "CSI_MODE",
									Value: "node",
								},{
									Name: "X_CSI_PERSISTENCE_CONFIG",
									Value: persistenceConfig,
									//Value: "{\"storage\":\"crd\"}",
								},{
									Name: "X_CSI_BACKEND_CONFIG",
									Value: backendConfig,
								},{
									Name: "X_CSI_NODE_ID",
									ValueFrom:  &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									MountPath: "/csi-data", 
									Name: "socket-dir",
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
								},{
									MountPath: "/var/lib/origin/openshift.local.volumes", 
									Name: "mountpoint-dir",
									MountPropagation: &bidirectional,
								},{
									MountPath: "/var/lib/kubelet/device-plugins", 
									Name: "kubelet-socket-dir",
									MountPropagation: &bidirectional,
								},{
									MountPath: "/etc/ceph", 
									Name: "config",
								},
							},
						},
					},
                                        Volumes: []v1.Volume{
						{
                                                        Name: "socket-dir",
                                                        VolumeSource: v1.VolumeSource{
                                                                HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins/io.ember-csi",
									Type: &dirOrCreate,
								},
                                                        },
						},{
                                                        Name: "mountpoint-dir",
                                                        VolumeSource: v1.VolumeSource{
                                                                HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/origin/openshift.local.volumes",
								},
                                                        },
                                                },{
                                                        Name: "kubelet-socket-dir",
                                                        VolumeSource: v1.VolumeSource{
                                                                HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/device-plugins",
								},
                                                        },
                                                },{
                                                        Name: "run-dir",
                                                        VolumeSource: v1.VolumeSource{
                                                                HostPath: &v1.HostPathVolumeSource{
									Path: "/run/udev",
								},
                                                        },
                                                },{
                                                        Name: "dev-dir",
                                                        VolumeSource: v1.VolumeSource{
                                                                HostPath: &v1.HostPathVolumeSource{
									Path: "/dev",
								},
                                                        },
                                                },{
                                                        Name: "localtime",
                                                        VolumeSource: v1.VolumeSource{
                                                                HostPath: &v1.HostPathVolumeSource{
									Path: "/etc/localtime",
								},
                                                        },
                                                },{
							Name: "config",
								VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{Name:configMapName},
								},
							},
						},
					},
				},
			},

		},
	}

        addOwnerRefToObject(ds, asOwner(ecsi))
	return ds
}
