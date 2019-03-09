package embercsi

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        appsv1 "k8s.io/api/apps/v1"
        "fmt"
)

// daemonSetForEmberCSI returns a EmberCSI DaemonSet object
func (r *ReconcileEmberCSI) daemonSetForEmberCSI(ecsi *embercsiv1alpha1.EmberCSI) *appsv1.DaemonSet {
	ls := labelsForEmberCSI(ecsi.Name)

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
                        Template: corev1.PodTemplateSpec{
                                ObjectMeta: metav1.ObjectMeta{
                                        Labels: ls,
                                },
                                Spec: corev1.PodSpec{
					ServiceAccountName: NodeSA,
					HostNetwork: true,
					HostIPC: true,
                                        Containers: getNodeContainers(ecsi),
                                        Volumes: generateVolumes(ecsi, "node"),
				},
			},
		},
	}
	controllerutil.SetControllerReference(ecsi, ds, r.scheme)

	return ds
}

// Construct a Containers PodSpec for Nodes
func getNodeContainers(ecsi *embercsiv1alpha1.EmberCSI) []corev1.Container {
	trueVar 		:= true
	probeHandler := corev1.Handler{
		Exec: &corev1.ExecAction{
			Command: []string{ "ember-liveness", },
		},
	}

        livenessProbe := &corev1.Probe{
		Handler:          probeHandler,
        }

	containers := []corev1.Container {
				{
					Name:    		"ember-csi-driver",
					Image:   		Conf.getDriverImage(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG),
					ImagePullPolicy: 	corev1.PullAlways,
					SecurityContext: 	&corev1.SecurityContext{
									Privileged: &trueVar,
									AllowPrivilegeEscalation: &trueVar,
								},
					TerminationMessagePath: "/tmp/termination-log",
					Env: 			generateEnvVars(ecsi, "node"),
					VolumeMounts: 		generateVolumeMounts(ecsi, "node"),
					LivenessProbe:		livenessProbe,
				},
			}

	// Add NodeRegistrar sidecar
	if len(Conf.Sidecars[Cluster].NodeRegistrar) > 0 {
		containers = append(containers, corev1.Container {
				Name:    "node-driver-registrar",
				Image:   Conf.Sidecars[Cluster].NodeRegistrar,
				Args: []string{ 
						"--v=5", 
						"--csi-address=/csi-data/csi.sock",
						fmt.Sprintf("%s/%s/%s", "--kubelet-registration-path=/var/lib/kubelet/plugins", GetPluginDomainName(ecsi.Name), "csi.sock"),
					},
				SecurityContext: &corev1.SecurityContext{ Privileged: &trueVar, },
				Env:	[]corev1.EnvVar{
					{
						Name: "KUBE_NODE_NAME",
						ValueFrom:  &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "spec.nodeName",
							},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						MountPath: "/csi-data",
						Name: "socket-dir",
					},
				},
			},
		)
	}

	// On older CSI specs, use driver registrar
	if len(Conf.Sidecars[Cluster].Registrar) > 0 {
		containers = append(containers, corev1.Container {
				Name:    "driver-registrar",
				Image:   Conf.Sidecars[Cluster].Registrar,
				Args: []string{ 
						"--v=5",
						"--csi-address=/csi-data/csi.sock",
					},
				SecurityContext: &corev1.SecurityContext{ Privileged: &trueVar, },
				Env:	[]corev1.EnvVar{
					{
						Name: "KUBE_NODE_NAME",
						ValueFrom:  &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "spec.nodeName",
							},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						MountPath: "/csi-data",
						Name: "socket-dir",
					},
				},
			},
		)
	}
		
	return containers
}

