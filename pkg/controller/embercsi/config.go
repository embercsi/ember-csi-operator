package embercsi

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"os"
	"strconv"
	"strings"
)

type EmberCSIOperatorConfig struct {
	viper   *viper.Viper
	cluster string
}

var k8sClientset *kubernetes.Clientset

// Return CSI Spec Version as a Float for comparison
func (emberCSIOperatorConfig EmberCSIOperatorConfig) getCSISpecVersion() float64 {
	csiSpecVersion := emberCSIOperatorConfig.viper.GetString(fmt.Sprintf("sidecars.%s.X_CSI_SPEC_VERSION", emberCSIOperatorConfig.cluster))
	if strings.HasPrefix(csiSpecVersion, "v") {
		csiSpecVersion = strings.Replace(csiSpecVersion, "v", "", -1)
	}

	spec, err := strconv.ParseFloat(csiSpecVersion, 64)
	if err != nil {
		glog.Info(fmt.Sprintf("Could't convert X_CSI_SPEC_VERSION to float. Using default: %f", DEFAULT_CSI_SPEC))
		spec = DEFAULT_CSI_SPEC // Use our sane default
	}
	return spec
}

// Get sidecar image for the corresponding sidecar
func (emberCSIOperatorConfig EmberCSIOperatorConfig) getSidecarImage(sidecarName string) string {
	return emberCSIOperatorConfig.viper.GetString(fmt.Sprintf("sidecars.%s.%s", emberCSIOperatorConfig.cluster, sidecarName))
}

// Return CSI Spec Version with the prefix 'v', if present
func (emberCSIOperatorConfig EmberCSIOperatorConfig) getRawCSISpecVersion() string {
	return emberCSIOperatorConfig.viper.GetString(fmt.Sprintf("sidecars.%s.X_CSI_SPEC_VERSION", emberCSIOperatorConfig.cluster))
}

// Get Ember CSI driver image
func (emberCSIOperatorConfig EmberCSIOperatorConfig) getDriverImage(backend string) string {
	return emberCSIOperatorConfig.viper.GetString(fmt.Sprintf("drivers.%s", backend))
}

// Configure and Initialize Viper to read our config
func ProcessConfig() {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(ConfigLocation) // Location is /etc/ember-csi-operator/config.yaml

	err := v.ReadInConfig()
	if err != nil {
		//glog.Fatal("Error. Unable to read config file. Using Default")
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var clusterID string
	// Read X_EMBER_OPERATOR_CLUSTER e.g ocp-3.11, k8s-1.13, k8s-1.14, etc
	if len(os.Getenv("X_EMBER_OPERATOR_CLUSTER")) > 0 {
		clusterID = os.Getenv("X_EMBER_OPERATOR_CLUSTER")
	} else {
		clusterID = "default"
	}

	// Watch changes and Act accordingly
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		glog.Infof("INFO: Config changed detected. Sync Required: %s", e.Name)
	})

	emberCSIOperatorConfig = &EmberCSIOperatorConfig{
		viper:   v,
		cluster: clusterID,
	}
}
