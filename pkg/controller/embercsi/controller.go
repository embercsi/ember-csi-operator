package embercsi

import (
	"fmt"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// csiDriverForEmberStorageBackend returns a EmberStorageBackend CSIDriver object
func (r *ReconcileEmberStorageBackend) csiDriverForEmberStorageBackend(ecsi *embercsiv1alpha1.EmberStorageBackend) *storagev1.CSIDriver {
	trueVar := true

	driver := &storagev1.CSIDriver{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "storage.k8s.io/v1",
			Kind:       "CSIDriver",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetPluginDomainName(ecsi.Name),
		},
		Spec: storagev1.CSIDriverSpec{
			PodInfoOnMount: &trueVar,
			AttachRequired: &trueVar,
		},
	}
	controllerutil.SetControllerReference(ecsi, driver, r.scheme)
	return driver
}



// statefulSetForEmberStorageBackend returns a EmberStorageBackend StatefulSet object
func (r *ReconcileEmberStorageBackend) statefulSetForEmberStorageBackend(ecsi *embercsiv1alpha1.EmberStorageBackend) *appsv1.StatefulSet {
	ls := labelsForEmberStorageBackend(ecsi.Name)

	// There *must* only be one replica
	var replicas int32 = 1

	ss := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-controller", ecsi.Name),
			Namespace: ecsi.Namespace,
			Labels:    ls,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers:         getControllerContainers(ecsi),
					Volumes:            generateVolumes(ecsi, "controller"),
					ServiceAccountName: ControllerSA,
					NodeSelector:       ecsi.Spec.NodeSelector,
					Tolerations:        ecsi.Spec.Tolerations,
					HostNetwork:        true,
					HostIPC:            true,
					HostPID:            true,
				},
			},
		},
	}
	controllerutil.SetControllerReference(ecsi, ss, r.scheme)
	return ss
}

// Construct a Containers PodSpec for Controller
func getControllerContainers(ecsi *embercsiv1alpha1.EmberStorageBackend) []corev1.Container {
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
			Env:          generateEnvVars(ecsi, "controller"),
			VolumeMounts: generateVolumeMounts(ecsi, "controller"),
			//	LivenessProbe:		livenessProbe,
			EnvFrom: []corev1.EnvFromSource{{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("ember-csi-operator-%s", ecsi.Name),
                                        },
                                },
			}},
		},
	}

	// Add External Attacher sidecar
	if len(Conf.Sidecars[Cluster].Attacher) > 0 {
		containers = append(containers, corev1.Container{
			Name:  "external-attacher",
			Image: Conf.Sidecars[Cluster].Attacher,
			Args: []string{"--v=5",
				"--csi-address=/csi-data/csi.sock",
				"--timeout=120s",
			},
			SecurityContext: &corev1.SecurityContext{Privileged: &trueVar},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/csi-data",
					Name:      "socket-dir",
				},
			},
		},
		)
	}

	// Add External Provisioner sidecar
	if len(Conf.Sidecars[Cluster].Provisioner) > 0 {
		// Customize the arguments for the container
		args := []string{
			"--v=5",
			"--csi-address=/csi-data/csi.sock",
		}
		if Conf.getCSISpecVersion() < 1.0 {
			args = append(args, fmt.Sprintf("%s%s", "--provisioner=", GetPluginDomainName(ecsi.Name)))
		}
		if Conf.getCSISpecVersion() > 0.3 {
			args = append(args, "--feature-gates=Topology=true")
		}
		containers = append(containers, corev1.Container{
			Name:            "external-provisioner",
			Image:           Conf.Sidecars[Cluster].Provisioner,
			Args:            args,
			SecurityContext: &corev1.SecurityContext{Privileged: &trueVar},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/csi-data",
					Name:      "socket-dir",
				},
			},
		},
		)
	}

	// Add ClusterRegistrar sidecar
	if len(Conf.Sidecars[Cluster].ClusterRegistrar) > 0 {
		containers = append(containers, corev1.Container{
			Name:  "cluster-driver-registrar",
			Image: Conf.Sidecars[Cluster].ClusterRegistrar,
			Args: []string{
				"--csi-address=/csi-data/csi.sock",
			},
			SecurityContext: &corev1.SecurityContext{Privileged: &trueVar},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/csi-data",
					Name:      "socket-dir",
				},
			},
		},
		)
	}

	// Add Snapshotter sidecar
	if len(Conf.Sidecars[Cluster].Snapshotter) > 0 {
		containers = append(containers, corev1.Container{
			Name:  "external-snapshotter",
			Image: Conf.Sidecars[Cluster].Snapshotter,
			Args: []string{"--v=5",
				"--csi-address=/csi-data/csi.sock",
			},
			SecurityContext: &corev1.SecurityContext{Privileged: &trueVar},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/csi-data",
					Name:      "socket-dir",
				},
			},
		},
		)
	}

	// Add External Resizer sidecar
	if len(Conf.Sidecars[Cluster].Resizer) > 0 {
		// Customize the arguments for the container
		args := []string{
			"--v=5",
			"--csi-address=/csi-data/csi.sock",
		}

		containers = append(containers, corev1.Container{
			Name:            "external-resizer",
			Image:           Conf.Sidecars[Cluster].Resizer,
			Args:            args,
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
