package embercsi

import (
	"fmt"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"encoding/json"
)

// daemonSetForEmberStorageBackend returns a EmberStorageBackend DaemonSet object
func (r *ReconcileEmberStorageBackend) daemonSetForEmberStorageBackend(ecsi *embercsiv1alpha1.EmberStorageBackend, daemonSetIndex int) *appsv1.DaemonSet {
	newEcsi := ecsi.DeepCopy()
	ls := labelsForEmberStorageBackend(ecsi.Name)

	if len(ecsi.Spec.Topologies) > 0 { // DaemonSet with specified topology

		var nodeSelectorRequirement []corev1.NodeSelectorRequirement
		//var nodeSelectorOperator corev1.NodeSelectorOperator
		if daemonSetIndex >= 1 {
			nodeSelectorRequirement = ecsi.Spec.Topologies[daemonSetIndex-1].Nodes
		} else { // Index == 0
			nodeSelectorRequirement = getNodesWithTopologies(newEcsi)

			// Invert the Operator to create an antiaffinity
			for index, key := range nodeSelectorRequirement {
				if key.Operator == corev1.NodeSelectorOpDoesNotExist {
					nodeSelectorRequirement[index].Operator = corev1.NodeSelectorOpExists
				}
				if key.Operator == corev1.NodeSelectorOpExists {
					nodeSelectorRequirement[index].Operator = corev1.NodeSelectorOpDoesNotExist
				}
				if key.Operator == corev1.NodeSelectorOpIn {
					nodeSelectorRequirement[index].Operator = corev1.NodeSelectorOpNotIn
				}
				if key.Operator == corev1.NodeSelectorOpNotIn {
					nodeSelectorRequirement[index].Operator = corev1.NodeSelectorOpIn
				}
				if key.Operator == corev1.NodeSelectorOpGt {
					nodeSelectorRequirement[index].Operator = corev1.NodeSelectorOpLt
				}
				if key.Operator == corev1.NodeSelectorOpLt {
					nodeSelectorRequirement[index].Operator = corev1.NodeSelectorOpGt
				}
			}

		}

		ds := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "DaemonSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-node-%d", ecsi.Name, daemonSetIndex),
				Namespace: ecsi.Namespace,
				Labels:    ls,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: ls,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: ls,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: NodeSA,
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{
										{
											MatchExpressions: nodeSelectorRequirement,
										},
									},
								},
							},
						},
						HostNetwork: true,
						HostIPC:     true,
						HostPID:     true,
						Containers:  getNodeContainers(ecsi, daemonSetIndex),
						Volumes:     generateVolumes(ecsi, "node"),
					},
				},
			},
		} // end-&appsv1.DaemonSet

		controllerutil.SetControllerReference(ecsi, ds, r.scheme)
		return ds
	} // end-if

	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-node-0", ecsi.Name), // 0 will be the default daemonSet's index
			Namespace: ecsi.Namespace,
			Labels:    ls,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: NodeSA,
					HostNetwork:        true,
					HostIPC:            true,
					HostPID:            true,
					Containers:         getNodeContainers(ecsi, 0),
					Volumes:            generateVolumes(ecsi, "node"),
				},
			},
		},
	} // end-&appsv1.DaemonSet

	controllerutil.SetControllerReference(ecsi, ds, r.scheme)
	return ds
}

// Construct a Containers PodSpec for Nodes
func getNodeContainers(ecsi *embercsiv1alpha1.EmberStorageBackend, daemonSetIndex int) []corev1.Container {
	trueVar := true

	containers := []corev1.Container{
		{
			Name:            "ember-csi",
			Image:           Conf.getDriverImage(ecsi.Spec.Config),
			ImagePullPolicy: corev1.PullAlways,
			SecurityContext: &corev1.SecurityContext{
				Privileged:               &trueVar,
				AllowPrivilegeEscalation: &trueVar,
			},
			TerminationMessagePath: "/tmp/termination-log",
			Env:          generateNodeEnvVars(ecsi, daemonSetIndex),
			VolumeMounts: generateVolumeMounts(ecsi, "node"),
			//LivenessProbe:		livenessProbe,
		},
	}

	// Add NodeRegistrar sidecar
	if len(Conf.Sidecars[Cluster].NodeRegistrar) > 0 {
		containers = append(containers, corev1.Container{
			Name:  "node-driver-registrar",
			Image: Conf.Sidecars[Cluster].NodeRegistrar,
			Args: []string{
				"--v=5",
				"--csi-address=/csi-data/csi.sock",
				fmt.Sprintf("%s/%s/%s", "--kubelet-registration-path=/var/lib/kubelet/plugins", GetPluginDomainName(ecsi.Name), "csi.sock"),
			},
			SecurityContext: &corev1.SecurityContext{Privileged: &trueVar},
			Env: []corev1.EnvVar{
				{
					Name: "KUBE_NODE_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/csi-data",
					Name:      "socket-dir",
				},
				{
					MountPath: "/registration",
					Name:      "registration-dir",
				},
			},
		},
		)
	}

	// On older CSI specs, use driver registrar
	if len(Conf.Sidecars[Cluster].Registrar) > 0 {
		containers = append(containers, corev1.Container{
			Name:  "driver-registrar",
			Image: Conf.Sidecars[Cluster].Registrar,
			Args: []string{
				"--v=5",
				"--csi-address=/csi-data/csi.sock",
			},
			SecurityContext: &corev1.SecurityContext{Privileged: &trueVar},
			Env: []corev1.EnvVar{
				{
					Name: "KUBE_NODE_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/csi-data",
					Name:      "socket-dir",
				},
			},
		},
		)
	}

	return containers
}

// construct EnvVars for the Driver Pod
func generateNodeEnvVars(ecsi *embercsiv1alpha1.EmberStorageBackend, daemonSetIndex int) []corev1.EnvVar {

	envVars := []corev1.EnvVar{
		{
			Name:  "PYTHONUNBUFFERED",
			Value: "0",
		}, {
			Name:  "CSI_ENDPOINT",
			Value: "unix:///csi-data/csi.sock",
		}, {
			Name:  "X_CSI_SPEC_VERSION",
			Value: Conf.Sidecars[Cluster].CSISpecVersion,
		}, {
			Name: "X_CSI_NODE_ID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		}, {
			Name:  "CSI_MODE",
			Value: "node",
		},
	}

	X_CSI_EMBER_CONFIG, err := interfaceToString(ecsi.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG)
	if err == nil {
		setJsonKeyIfEmpty(&X_CSI_EMBER_CONFIG, "plugin_name", GetPluginDomainName(ecsi.Name))
		setJsonKeyIfEmpty(&X_CSI_EMBER_CONFIG, "project_id", "ember-csi.io")
		setJsonKeyIfEmpty(&X_CSI_EMBER_CONFIG, "user_id", "ember-csi.io")
		setJsonKeyIfEmpty(&X_CSI_EMBER_CONFIG, "root_helper", "sudo")

		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_EMBER_CONFIG",
			Value: X_CSI_EMBER_CONFIG,
		})
	} else {
		glog.Errorf("Error parsing X_CSI_EMBER_CONFIG: %v\n", err)
	}

	X_CSI_PERSISTENCE_CONFIG, err := interfaceToString(ecsi.Spec.Config.EnvVars.X_CSI_PERSISTENCE_CONFIG)
	if err == nil {
		setJsonKeyIfEmpty(&X_CSI_PERSISTENCE_CONFIG, "storage", "crd")
		setJsonKeyIfEmpty(&X_CSI_PERSISTENCE_CONFIG, "namespace", ecsi.Namespace)
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_PERSISTENCE_CONFIG",
			Value: X_CSI_PERSISTENCE_CONFIG,
		})
	} else {
		glog.Errorf("Error parsing X_CSI_PERSISTENCE_CONFIG: %v\n", err)
	}

	if len(ecsi.Spec.Config.EnvVars.X_CSI_DEBUG_MODE) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_DEBUG_MODE",
			Value: ecsi.Spec.Config.EnvVars.X_CSI_DEBUG_MODE,
		},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_ABORT_DUPLICATES) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_ABORT_DUPLICATES",
			Value: ecsi.Spec.Config.EnvVars.X_CSI_ABORT_DUPLICATES,
		},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_DEFAULT_MOUNT_FS) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_DEFAULT_MOUNT_FS",
			Value: ecsi.Spec.Config.EnvVars.X_CSI_DEFAULT_MOUNT_FS,
		},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_NODE_ID) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_NODE_ID",
			Value: ecsi.Spec.Config.EnvVars.X_CSI_NODE_ID,
		},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.X_CSI_STORAGE_NW_IP) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_STORAGE_NW_IP",
			Value: ecsi.Spec.Config.EnvVars.X_CSI_STORAGE_NW_IP,
		},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.CSI_ENDPOINT) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "CSI_ENDPOINT",
			Value: ecsi.Spec.Config.EnvVars.CSI_ENDPOINT,
		},
		)
	}
	if len(ecsi.Spec.Config.EnvVars.CSI_MODE) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "CSI_MODE",
			Value: ecsi.Spec.Config.EnvVars.CSI_MODE,
		},
		)
	}
	if len(ecsi.Spec.Config.SysFiles.Name) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_SYSTEM_FILES",
			Value: fmt.Sprintf("/tmp/ember-csi/%s", ecsi.Spec.Config.SysFiles.Key),
		},
		)
	}
	// Topology enabled
	if len(ecsi.Spec.Topologies) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_NODE_TOPOLOGY",
			Value: getTopology(ecsi, daemonSetIndex),
		},
		)
	}

	return envVars
}

// Fetch topology based on index
func getTopology(ecsi *embercsiv1alpha1.EmberStorageBackend, index int) string {
	// Default topology
	defaultTopology := fmt.Sprintf("{\"%s-%s\": \"%s\"}", GetPluginDomainName(ecsi.Name), "csi-topology", "not-used")

	// Topology is specified but we are default daemonSet
	if index == 0 {
		glog.Infof("Using default topology: %s\n", defaultTopology)
		return defaultTopology
	}

	jsonBytes := []byte("{}")
	if len(ecsi.Spec.Topologies[index-1].Topology) > 0 {
		jsonBytes, _ = json.Marshal(ecsi.Spec.Topologies[index-1].Topology)
	}

	glog.Infof("Using topology for daemonSet: node-%d : %s\n", index, string(jsonBytes))
	return string(jsonBytes)
}

// Fetch all nodes with topologies
func getNodesWithTopologies(ecsi *embercsiv1alpha1.EmberStorageBackend) []corev1.NodeSelectorRequirement {
	var nodesWithTopologies []corev1.NodeSelectorRequirement

	if len(ecsi.Spec.Topologies) > 0 {
		// Create a daemonSet for each allowed topology
		for _, topologyItem := range ecsi.Spec.Topologies {
			nodesWithTopologies = append(nodesWithTopologies, topologyItem.Nodes...)
		}
	}

	return nodesWithTopologies
}
