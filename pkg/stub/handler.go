package stub

import (
	"context"

	"github.com/kirankt/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"

	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
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

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.EmberCSI:

		embercsi := o

		// Ignore the delete event since the garbage collector will clean up all secondary resources for the CR
		// All secondary resources must have the CR set as their OwnerReference for this to be the case
		if event.Deleted {
			return nil
		}

		logrus.Infof("Creating StatefulSet for Ember CSI Controller")

		// Create the Controller StatefulSet if it doesn't exist
		ss := statefulsetForEmberCSI(embercsi)
		err := sdk.Create(ss)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create statefulset: %v", err)
		}

		// Ensure the statefulset size is the same as the spec
		err = sdk.Get(ss)
		if err != nil {
			return fmt.Errorf("failed to get StatefulSet: %v", err)
		}
		size := embercsi.Spec.Size
		if *ss.Spec.Replicas != size {
			*ss.Spec.Replicas = size
			err = sdk.Update(ss)
			if err != nil {
				return fmt.Errorf("failed to update StatefulSet: %v", err)
			}
		}

	}
	return nil
}

// statefulsetForEmberCSI returns a EmberCSI StatefulSet object
func statefulsetForEmberCSI(ecsi *v1alpha1.EmberCSI) *appsv1.StatefulSet {
	ls := labelsForEmberCSI(ecsi.Name)

	// There *must* only be one replica
	var replicas int32 = 1

	secretName := ecsi.Spec.Secret

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
						Image:   "quay.io/k8scsi/csi-attacher:v0.3.0",
						Name:    "external-attacher",
						Args: []string{"--v=5", "--csi-address=/csi-data/csi.sock"},
					},{
						Image:   "quay.io/k8scsi/csi-provisioner:v0.3.0",
						Name:    "external-provisioner",
						Args: []string{"--v=5", "--csi-address=/csi-data/csi.sock", "--provisioner=io.ember-csi"},
					},{
						Image:   "akrog/ember-csi:master",
						Name:    "ember-csi-driver",
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
								Value: "{\"storage\":\"crd\"}",
							},{
								Name: "X_CSI_BACKEND_CONFIG",
								ValueFrom: &v1.EnvVarSource{
									SecretKeyRef: &v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: secretName},
										Key:  "backend_config",
									},
								},
							},{
								Name: "X_CSI_BACKEND_CONFIG",
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
								HostPath: &v1.HostPathVolumeSource{},
							},
						},{
							Name: "localtime",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{},
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

// serviceAccountForDS returns a ServiceAccount used by Ember CSI DaemonSet
func serviceAccountForDS(ecsi *v1alpha1.EmberCSI) *v1.ServiceAccount {
	dsa := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
                        Name:      fmt.Sprintf("%s-node-sa", ecsi.Name),
                        Namespace: ecsi.Namespace,
		},
	}
	return dsa
}

// daemonSetForEmberCSI returns a EmberCSI DaemonSet object
func daemonSetForEmberCSI(ecsi *v1alpha1.EmberCSI) *appsv1.DaemonSet {
	ls := labelsForEmberCSI(ecsi.Name)

	var dirOrCreate v1.HostPathType = v1.HostPathDirectoryOrCreate

	secretName := ecsi.Spec.Secret

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
                                        Containers: []v1.Container{{
                                                Image:   "quay.io/k8scsi/driver-registrar:v0.3.0",
                                                Name:    "driver-registrar",
                                                Args: []string{"--v=5", "--csi-address=/csi-data/csi.sock"},
                                        },{
                                                Image:   "akrog/ember-csi:master",
                                                Name:    "ember-csi-driver",
                                                Env: []v1.EnvVar{
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
                                                                Value: "{\"storage\":\"crd\"}",
                                                        },{
                                                                Name: "X_CSI_BACKEND_CONFIG",
                                                                ValueFrom: &v1.EnvVarSource{
                                                                        SecretKeyRef: &v1.SecretKeySelector{
                                                                                LocalObjectReference: v1.LocalObjectReference{Name: secretName},
                                                                                Key:  "backend_config",
                                                                        },
                                                                },
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
                                                                MountPath: "/csi-data", Name: "socket-dir",
                                                        },{
                                                                MountPath: "/dev", Name: "dev-dir",
                                                        },{
                                                                MountPath: "/etc/localtime", Name: "localtime",
                                                        },{
                                                                MountPath: "/var/lib/origin/openshift.local.volumes", Name: "mountpoint-dir",
                                                        },{
                                                                MountPath: "/var/lib/kubelet/device-plugins", Name: "kubelet-socket-dir",
                                                        },
                                                },
                                        }},
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
                                                },
					},
				},
			},

		},
	}

        addOwnerRefToObject(ds, asOwner(ecsi))
	return ds
}
