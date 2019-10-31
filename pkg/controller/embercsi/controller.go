package embercsi

import (
	"fmt"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// statefulSetForEmberCSI returns a EmberCSI StatefulSet object
func (r *ReconcileEmberCSI) statefulSetForEmberCSI(ecsi *embercsiv1alpha1.EmberCSI) *appsv1.StatefulSet {
	ls := labelsForEmberCSI(ecsi.Name)

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
				},
			},
		},
	}
	controllerutil.SetControllerReference(ecsi, ss, r.scheme)
	return ss
}

// Construct a Containers PodSpec for Controller
func getControllerContainers(ecsi *embercsiv1alpha1.EmberCSI) []corev1.Container {
	trueVar := true

	containers := []corev1.Container{
		{
			Name:  "ember-csi-driver",
			Image: emberCSIOperatorConfig.getDriverImage(getBackendName(ecsi)),
			ImagePullPolicy: corev1.PullAlways,
			SecurityContext: &corev1.SecurityContext{
				Privileged:               &trueVar,
				AllowPrivilegeEscalation: &trueVar,
			},
			TerminationMessagePath: "/tmp/termination-log",
			Env:                    generateEnvVars(ecsi, "controller"),
			VolumeMounts:           generateVolumeMounts(ecsi, "controller"),
			//	LivenessProbe:		livenessProbe,
		},
	}

	// Add External Attacher sidecar
	if len(emberCSIOperatorConfig.getSidecarImage("external-attacher")) > 0 {
		containers = append(containers, corev1.Container{
			Name:  "external-attacher",
			Image: emberCSIOperatorConfig.getSidecarImage("external-attacher"),
			Args: []string{"--v=5",
				"--csi-address=/csi-data/csi.sock",
				"--connection-timeout=120s",
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
	if len(emberCSIOperatorConfig.getSidecarImage("external-provisioner")) > 0 {
		// Customize the arguments for the container
		args := []string{
			"--v=5",
			"--csi-address=/csi-data/csi.sock",
			fmt.Sprintf("%s%s", "--provisioner=", GetPluginDomainName(ecsi.Name)),
		}

		if emberCSIOperatorConfig.getCSISpecVersion() > 0.3 {
			args = append(args, "--feature-gates=Topology=true")
		}
		containers = append(containers, corev1.Container{
			Name:  "external-provisioner",
			Image: emberCSIOperatorConfig.getSidecarImage("external-provisioner"),
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
	if len(emberCSIOperatorConfig.getSidecarImage("cluster-driver-registrar")) > 0 {
		containers = append(containers, corev1.Container{
			Name:  "cluster-driver-registrar",
			Image: emberCSIOperatorConfig.getSidecarImage("cluster-driver-registrar"),
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
	if len(emberCSIOperatorConfig.getSidecarImage("external-snapshotter")) > 0 {
		containers = append(containers, corev1.Container{
			Name:  "external-snapshotter",
			Image: emberCSIOperatorConfig.getSidecarImage("external-snapshotter"),
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

	return containers
}
