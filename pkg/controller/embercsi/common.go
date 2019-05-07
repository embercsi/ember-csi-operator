package embercsi

import (
	"fmt"
	"strings" 
	"bytes"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/golang/glog"
)

// Default values
const (
        NodeSA string           = "ember-csi-operator"
        ControllerSA string     = "ember-csi-operator"
	DEFAULT_CSI_SPEC 	= 0.2
)

// Global variables
var Conf 	*Config
var Cluster 	string

// Plugin's domain name to use. Prior to CSI spec 1.0, we used reverse
// domain name, after 1.0 we use forward.
func GetPluginDomainName(instanceName string) string {
	if Conf.getCSISpecVersion() < 1.0 {
		glog.Info("CSI Spec is < 1.0 using reverse domain plugin name")
		return fmt.Sprintf("%s.%s", "io.ember-csi", instanceName)
	}
	glog.Info("CSI Spec is >= 1.0 using forward domain plugin name")
	return fmt.Sprintf("%s.%s", instanceName, "ember-csi.io")
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
                        Value: fmt.Sprintf("%s%s%s", "{\"plugin_name\": \"", GetPluginDomainName(ecsi.Name), "\", \"project_id\": \"io.ember-csi\", \"user_id\": \"io.ember-csi\", \"root_helper\": \"sudo\", \"request_multipath\": \"true\" }"),
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
		// Topologies are enabled
		if len(ecsi.Spec.Topologies) > 0 {
			envVars = append(envVars, corev1.EnvVar{
					Name: "X_CSI_TOPOLOGIES",
					Value: getAllCSITopologies(ecsi),
				},
			)
		}
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

// Return all accessible topologies known to the CSI Driver
// A json array of topologies is returned to be consumed by
// X_CSI_TOPOLOGIES
func getAllCSITopologies(ecsi *embercsiv1alpha1.EmberCSI) string {
	glog.V(3).Info("Creating all known allowed_topologies array")

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "[")
	for _, topologyItem := range ecsi.Spec.Topologies {
		fmt.Fprintf(&buf, "{")
		for topology, value := range topologyItem.Topology {
			fmt.Fprintf(&buf, "\"%s\":\"%s\",", topology, value)
		}
		buf.Truncate(buf.Len() - 1)     // Remove trailing ','
		fmt.Fprintf(&buf, "},")
	}
	buf.Truncate(buf.Len() - 1)     // Remove trailing ','
	fmt.Fprintf(&buf, "]")

	return buf.String()
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
		// Add NodeRegistrar sidecar
		if len(Conf.Sidecars[Cluster].NodeRegistrar) > 0 {
			vol = append(vol, corev1.Volume{
					Name: "registration-dir",
					VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/lib/kubelet/plugins_registry",
							},
					},
				},
			)
		}

		vol = append(vol, corev1.Volume{
				Name: "socket-dir",
				VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: fmt.Sprintf("%s/%s", "/var/lib/kubelet/plugins", GetPluginDomainName(ecsi.Name)),

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
