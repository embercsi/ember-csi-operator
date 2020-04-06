package embercsi

import (
	"bytes"
	"fmt"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"encoding/json"
	"time"
	"math/rand"
)

// Default values
const (
	NodeSA           string = "ember-csi-operator"
	ControllerSA     string = "ember-csi-operator"
	DEFAULT_CSI_SPEC        = 0.2
)

// Global variables
var Conf *Config
var Cluster string

// Plugin's domain name to use. Prior to CSI spec 1.0, we used reverse
// domain name, after 1.0 we use forward.
func GetPluginDomainName(instanceName string) string {
	if Conf.getCSISpecVersion() < 1.0 {
		return fmt.Sprintf("%s.%s", "io.ember-csi", instanceName)
	}
	return fmt.Sprintf("%s.%s", instanceName, "ember-csi.io")
}

// construct EnvVars for the Driver Pod
func generateEnvVars(ecsi *embercsiv1alpha1.EmberCSI, driverMode string) []corev1.EnvVar {
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
		},
	}

	if driverMode == "controller" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "KUBE_NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		}, corev1.EnvVar{
			Name:  "CSI_MODE",
			Value: "controller",
		},
		)
		// Topologies are enabled
		if len(ecsi.Spec.Topologies) > 0 {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "X_CSI_TOPOLOGIES",
				Value: getAllCSITopologies(ecsi),
			},
			)
		}
	} else {
		envVars = append(envVars, corev1.EnvVar{
			Name: "X_CSI_NODE_ID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		}, corev1.EnvVar{
			Name:  "CSI_MODE",
			Value: "node",
		},
		)
	}
	X_CSI_EMBER_CONFIG, err := interfaceToString(ecsi.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG)
	if err == nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_EMBER_CONFIG",
			Value: X_CSI_EMBER_CONFIG,
		},
		)
	} else {
		glog.Errorf("Error parsing X_CSI_EMBER_CONFIG: %v\n", err)
	}
	X_CSI_BACKEND_CONFIG, err := interfaceToString(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG)
	if err == nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_BACKEND_CONFIG",
			Value: X_CSI_BACKEND_CONFIG,
		},
		)
	} else {
		glog.Errorf("Error parsing X_CSI_BACKEND_CONFIG: %v\n", err)
	}
	X_CSI_PERSISTENCE_CONFIG, err := interfaceToString(ecsi.Spec.Config.EnvVars.X_CSI_PERSISTENCE_CONFIG)
	if err == nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "X_CSI_PERSISTENCE_CONFIG",
			Value: X_CSI_PERSISTENCE_CONFIG,
		},
		)
	} else {
		glog.Errorf("Error parsing X_CSI_PERSISTENCE_CONFIG,: %v\n", err)
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
		buf.Truncate(buf.Len() - 1) // Remove trailing ','
		fmt.Fprintf(&buf, "},")
	}
	buf.Truncate(buf.Len() - 1) // Remove trailing ','
	fmt.Fprintf(&buf, "]")

	return buf.String()
}

// Construct a VolumeMount based on cluster type, secrets, etc
func generateVolumeMounts(ecsi *embercsiv1alpha1.EmberCSI, csiDriverMode string) []corev1.VolumeMount {
	var bidirectional corev1.MountPropagationMode = corev1.MountPropagationBidirectional
	var hostToContainer corev1.MountPropagationMode = corev1.MountPropagationHostToContainer

	vm := []corev1.VolumeMount{
		{
			MountPath:        "/csi-data",
			Name:             "socket-dir",
			MountPropagation: &bidirectional,
		}, {
			MountPath:        "/etc/iscsi",
			Name:             "iscsi-dir",
			MountPropagation: &bidirectional,
		}, {
			MountPath:        "/var/lib/iscsi",
			Name:             "var-lib-iscsi",
			MountPropagation: &bidirectional,
		}, {
			MountPath:        "/etc/multipath",
			Name:             "multipath-dir",
			MountPropagation: &bidirectional,
		}, {
			MountPath:        "/etc/multipath.conf",
			Name:             "multipath-conf",
			MountPropagation: &hostToContainer,
		}, {
			MountPath:        "/lib/modules",
			Name:             "modules-dir",
			MountPropagation: &hostToContainer,
		}, {
			MountPath:        "/run/udev",
			Name:             "run-dir",
			MountPropagation: &hostToContainer,
		}, {
			MountPath:        "/dev",
			Name:             "dev-dir",
			MountPropagation: &bidirectional,
		}, {
			MountPath:        "/etc/localtime",
			Name:             "localtime",
			MountPropagation: &hostToContainer,
		},
	}

	// Check to see if the volume driver is LVM
	X_CSI_BACKEND_CONFIG, err := interfaceToString(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG)
	if err == nil {
		if strings.Contains(strings.ToLower(X_CSI_BACKEND_CONFIG), "lvmvolume") {
			vm = append(vm, corev1.VolumeMount{
				Name:             "etc-lvm",
				MountPath:        "/etc/lvm",
				MountPropagation: &bidirectional,
			}, corev1.VolumeMount{
				Name:             "var-lock-lvm",
				MountPath:        "/var/lock/lvm",
				MountPropagation: &bidirectional,
			},
			)
		}
	} else {
		glog.Errorf("Error parsing X_CSI_BACKEND_CONFIG: %v\n", err)
	}

	// Check to see if the X_CSI_SYSTEM_FILES secret is present in the CR
	if len(ecsi.Spec.Config.SysFiles.Name) > 0 {
		vm = append(vm, corev1.VolumeMount{
			Name:      "system-files",
			MountPath: "/tmp/ember-csi",
		},
		)
	}

	if csiDriverMode == "node" {
		// Ember CSI shared lock directory to survive restarts
		vm = append(vm, corev1.VolumeMount{
			Name:             "shared-lock-dir",
			MountPath:        "/var/lib/ember-csi",
			MountPropagation: &bidirectional,
		},
		)

		// ocp 3.x
		if strings.Contains(Cluster, "ocp-3") {
			vm = append(vm, corev1.VolumeMount{
				Name:             "mountpoint-dir",
				MountPropagation: &bidirectional,
				MountPath:        "/var/lib/origin/openshift.local.volumes",
			}, corev1.VolumeMount{
				MountPath:        "/var/lib/kubelet/device-plugins",
				Name:             "kubelet-socket-dir",
				MountPropagation: &bidirectional,
			},
			)
		} else { // k8s, ocp >= 4.x
			vm = append(vm, corev1.VolumeMount{
				Name:             "mountpoint-dir",
				MountPropagation: &bidirectional,
				MountPath:        "/var/lib/kubelet",
			},
			)
		}
	}

	return vm
}

func generateVolumes(ecsi *embercsiv1alpha1.EmberCSI, csiDriverMode string) []corev1.Volume {
	var dirOrCreate corev1.HostPathType = corev1.HostPathDirectoryOrCreate

	vol := []corev1.Volume{
		{
			Name: "run-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/run/udev",
				},
			},
		}, {
			Name: "dev-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/dev",
				},
			},
		}, {
			Name: "iscsi-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/iscsi",
				},
			},
		}, {
			Name: "var-lib-iscsi",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/iscsi",
				},
			},
		}, {
			Name: "multipath-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/multipath",
				},
			},
		}, {
			Name: "multipath-conf",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/multipath.conf",
				},
			},
		}, {
			Name: "modules-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/lib/modules",
				},
			},
		}, {
			Name: "localtime",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/localtime",
				},
			},
		},
	}

	// Check to see if the volume driver is LVM
	X_CSI_BACKEND_CONFIG, err := interfaceToString(ecsi.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG)
	if err == nil {
		if strings.Contains(strings.ToLower(X_CSI_BACKEND_CONFIG), "lvmvolume") {
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
	} else {
		glog.Errorf("Error parsing X_CSI_BACKEND_CONFIG: %v\n", err)
	}

	// Check to see if the X_CSI_SYSTEM_FILES secret is present in the CR
	if len(ecsi.Spec.Config.SysFiles.Name) > 0 {
		vol = append(vol, corev1.Volume{
			Name: "system-files",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ecsi.Spec.Config.SysFiles.Name,
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
		}, corev1.Volume{
			Name: "shared-lock-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/ember-csi",
					Type: &dirOrCreate,
				},
			},
		},
		)
		// ocp 3.x
		if strings.Contains(Cluster, "ocp-3") {
			vol = append(vol, corev1.Volume{
				Name: "mountpoint-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/origin/openshift.local.volumes",
					},
				},
			}, corev1.Volume{
				Name: "kubelet-socket-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/kubelet/device-plugins",
						Type: &dirOrCreate,
					},
				},
			},
			)
		} else { // k8s, ocp >= 4.x
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
	} else { // "controller" or "all" mode
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

// Check whether feature is enabled/disabled
func isFeatureEnabled(emberConfig string, feature string) bool {
	type EmberCSIConfig struct {
		Disabled        []string
	}
	var ecc EmberCSIConfig

	err := json.Unmarshal([]byte(emberConfig), &ecc)
	if err != nil {
		glog.Warningf("Forwarding unmodified input %v (type %T) to Ember\n", emberConfig, emberConfig)
		glog.Error(err)
	}
	glog.V(3).Infof("Info: X_CSI_EMBER_CONFIG Disabled Features: %v", ecc.Disabled)

	for i := 0; i < len(ecc.Disabled); i++ {
                if ecc.Disabled[i] == feature {
			glog.V(3).Infof("Info: %s disabled in Ember config via X_CSI_EMBER_CONFIG", feature)
			return false
		}
	}
        return true
}


func interfaceToString(input interface{}) (string, error) {
	m, ok := input.(map[string]interface{})
	if ok {
		jsonString, _ := json.Marshal(m)
		return configTransform(string(jsonString)), nil
	}

	// String, maybe a JSON?
	s, ok := input.(string)
	if ok {
		j := make(map[string]interface{})
		err := json.Unmarshal([]byte(s), &j)
		if err == nil {
			jsonString, _ := json.Marshal(j)
			return configTransform(string(jsonString)), nil
		} else { // string, but not valid JSON
			err := fmt.Errorf("Forwarding unmodified input %v (type %T) to Ember\n", input, input)
			return s, err
		}
	}

	// Something else, fail safely
	err := fmt.Errorf("Could not marshal %v (type %T) to JSON\n", input, input)
	return "{}", err
}


func configTransform(input string) string {
	b := []byte(input)
	var m map[string]interface{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		glog.Warningf("Forwarding unmodified input %v (type %T) to Ember\n", input, input)
		glog.Error(err)
		return input
	}

	driver_prefix := "driver__"
	driver_prefix_keep := fmt.Sprintf("driver__%s__", m["driver"])

	for k, v := range m {
		if strings.HasSuffix(k, "__transform_empty_none") && v == "" {
			newkey := strings.Replace(k, "__transform_empty_none", "", -1)
			m[newkey] = nil
			delete(m, k)
		}

		if strings.HasSuffix(k, "__transform_csv") {
			newkey := strings.Replace(k, "__transform_csv", "", -1)
			m[newkey] = strings.Split(v.(string), ",")
			delete(m, k)
		}

		if strings.HasSuffix(k, "__transform_csv_kvs") {
			newkey := strings.Replace(k, "__transform_csv_kvs", "", -1)

			kvpairs := strings.Split(v.(string), ",")
			for _, pair := range kvpairs {
				s := strings.Split(pair, ":")
				if len(s) > 1 {
					subkey, subval := s[0], s[1]
					submap := make(map[string]string)
					submap[subkey] = subval
					m[newkey] = submap
				}
			}
			delete(m, k)
		}
	}

	// Need to iterate again, keys might have been transformed earlier
	for k, v := range m {
		if strings.HasPrefix(k, driver_prefix) {
			delete(m, k)
			if strings.HasPrefix(k, driver_prefix_keep) {
				newkey := strings.Replace(k, driver_prefix_keep, "", -1)
				m[newkey] = v
			}
		}
	}

	output, err := json.Marshal(m)
	if err != nil {
		glog.Warningf("Forwarding unmodified input %v (type %T) to Ember\n", input, input)
		return input
	}

	return string(output)
}


func setJsonKeyIfEmpty(input *string, key string, val string) string {
	b := []byte(*input)
	var m map[string]interface{}

	err := json.Unmarshal(b, &m)
	if err != nil {
		glog.Error(err)
		return *input
	}

	v, ok := m[key]
	if !ok || (ok && v == "") {
		m[key] = val
	}

	output, err := json.Marshal(m)
	if err != nil {
		glog.Error(err)
		return *input
	}
	*input = string(output)
	return string(output)
}


func randomString(length int) string {
    chars := []int32("abcdefghijklmnopqrstuvwxyz01234567890")
    rand.Seed(time.Now().UnixNano())
    a := make([]int32, length)
    for i := range a {
        a[i] = chars[rand.Intn(len(chars))]
    }
    return string(a)
}
