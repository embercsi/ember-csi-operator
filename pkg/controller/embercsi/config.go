package embercsi

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type Versions struct {
	CSISpecVersion   string `yaml:"X_CSI_SPEC_VERSION,omitempty"`
	Attacher         string `yaml:"external-attacher,omitempty"`
	Provisioner      string `yaml:"external-provisioner,omitempty"`
	Registrar        string `yaml:"driver-registrar,omitempty"` // For use in older CSI specs
	NodeRegistrar    string `yaml:"node-driver-registrar,omitempty"`
	ClusterRegistrar string `yaml:"cluster-driver-registrar,omitempty"`
	Resizer          string `yaml:"external-resizer,omitempty"`
	Snapshotter      string `yaml:"external-snapshotter,omitempty"`
	LivenessProbe    string `yaml:"livenessprobe,omitempty"`
}

type Config struct {
	ConfigVersion string              `yaml:"version,omitempty"`
	Sidecars      map[string]Versions `yaml:"sidecars,omitempty"`
	Drivers       map[string]string   `yaml:"drivers"`
}

func (config *Config) getDriverImage(backend_config string) string {
	var backend_config_map map[string]string
	json.Unmarshal([]byte(backend_config), &backend_config_map)
	backend := backend_config_map["driver"]
	var image string

	if len(backend) > 0 && len(config.Drivers[backend]) > 0 {
		image = config.Drivers[backend]
	} else if len(config.Drivers["default"]) > 0 {
		image = config.Drivers["default"]
	} else {
		image = "embercsi/ember-csi:master"
	}
	glog.Infof(fmt.Sprintf("Using driver image %s", image))
	return image
}

func (config *Config) getCluster() string {
	return Cluster
}

// Returns a float value of CSI_SPEC
func (config *Config) getCSISpecVersion() float64 {

	// Remove 'v' prefix if it exists
	if strings.HasPrefix(Conf.Sidecars[Cluster].CSISpecVersion, "v") { // starts with 'v' e.g. v0.3
		var tmpConf = Conf.Sidecars[Cluster]
		tmpConf.CSISpecVersion = strings.Replace(Conf.Sidecars[Cluster].CSISpecVersion, "v", "", -1)
		Conf.Sidecars[Cluster] = tmpConf
	}

	spec, err := strconv.ParseFloat(Conf.Sidecars[Cluster].CSISpecVersion, 64)
	if err != nil {
		glog.Info(fmt.Sprintf("Could't convert X_CSI_SPEC_VERSION to float. Using default: %f", DEFAULT_CSI_SPEC))
		// Use our sane default
		spec = DEFAULT_CSI_SPEC
	}
	return spec
}

// Read Config and store values from Config File or Use DefaultConfig
func ReadConfig(configFile *string) {
	// If configFile is not specified. Lets use our default
	if len(strings.TrimSpace(*configFile)) == 0 {
		*configFile = "/etc/ember-csi-operator/config.yaml"
	}

	source, err := ioutil.ReadFile(*configFile)
	if err != nil {
		glog.Infof("Cannot Open Config File: %s. Will use defaults.\n", *configFile)
		DefaultConfig()
	}
	err = yaml.Unmarshal(source, &Conf)
	if err != nil {
		glog.Info("Cannot Unmarshal Config File. Will use defaults.\n")
		DefaultConfig()
	}

	// Read X_EMBER_OPERATOR_CLUSTER e.g ocp-3.11, k8s-1.13, k8s-1.14, etc
	if len(os.Getenv("X_EMBER_OPERATOR_CLUSTER")) > 0 {
		Cluster = os.Getenv("X_EMBER_OPERATOR_CLUSTER")
	} else {
		// Use "default" as the cluster name which is used in DefaultConfig
		Cluster = "default"
	}
}

// Populate the Config Stuct with some default values and Return it
func DefaultConfig() {
	var defaultConfig = `
---
version: 1.0
sidecars:
  default:
    X_CSI_SPEC_VERSION: v0.3
    external-attacher: quay.io/k8scsi/csi-attacher:v0.3.0
    external-provisioner: quay.io/k8scsi/csi-provisioner:v0.3.0
    driver-registrar: quay.io/k8scsi/driver-registrar:v0.3.0
drivers:
  default: embercsi/ember-csi:master
`
	err := yaml.Unmarshal([]byte(defaultConfig), &Conf)
	if err != nil {
		glog.Fatal("Cannot Open Default Config", err)
	}
}
