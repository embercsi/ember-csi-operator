package embercsi

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"regexp"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/apimachinery/pkg/runtime/serializer"
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

func (config *Config) getDriverImage(spec_config embercsiv1alpha1.EmberStorageBackendConfig) string {
	backend_config, err := interfaceToString(spec_config.EnvVars.X_CSI_BACKEND_CONFIG)
	if err != nil {
		glog.Errorf("Error parsing X_CSI_BACKEND_CONFIG: %v\n", err)
	}

	var backend_config_map map[string]string
	json.Unmarshal([]byte(backend_config), &backend_config_map)
	backend := backend_config_map["driver"]
	image := spec_config.DriverImage

	if len(image) == 0 {
		if len(backend) > 0 && len(config.Drivers[backend]) > 0 {
			image = config.Drivers[backend]
		} else if len(config.Drivers["default"]) > 0 {
			image = config.Drivers["default"]
		} else {
			image = "embercsi/ember-csi:master"
		}
	}
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


func getClusterVersion() string {
	kubeconfig := flag.Lookup("kubeconfig").Value.(flag.Getter).Get().(string)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Error(err)
		return "default"
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Error(err)
		return "default"
	}


	discoveryClient := clientset.Discovery()
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		glog.Error(err)
		return "default"
	}
	parts := strings.Split(serverVersion.String(), ".")
	clusterVersion := fmt.Sprintf("k8s-%s", strings.Join(parts[0:2], "."))

	// Educated guess if this is an OpenShift cluster
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	config.APIPath = "/apis/config.openshift.io/v1/clusterversions"
	restClient, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		glog.Error(err)
		return clusterVersion
	}
	result, err := restClient.Get().Do(context.TODO()).Raw()
	if err == nil {
		// result is JSON, but there is no simple version key/val entry
		// Thus using a regexp to just match major.minor version
		re := regexp.MustCompile(`Cluster version is (\d+\.\d+)`)
		match := re.FindSubmatch(result)
		if len(match) > 0 {
			clusterVersion = fmt.Sprintf("ocp-%s", match[1])
		}
	}

	return clusterVersion
}


// Read Config and store values from Config File or Use DefaultConfig
func ReadConfig(configFile *string) {
	// If configFile is not specified. Lets use our default
	if len(strings.TrimSpace(*configFile)) == 0 {
		*configFile = "/etc/ember-csi-operator/config.yaml"
	}

	source, err := ioutil.ReadFile(*configFile)
	if err != nil {
		glog.Fatalf("Cannot Open Config File: %s\n", *configFile)
	}
	err = yaml.Unmarshal(source, &Conf)
	if err != nil {
		glog.Fatalf("Cannot Unmarshal Config File\n")
	}

	// Read X_EMBER_OPERATOR_CLUSTER e.g ocp-3.11, k8s-1.13, k8s-1.14, etc
	if len(os.Getenv("X_EMBER_OPERATOR_CLUSTER")) > 0 {
		Cluster = os.Getenv("X_EMBER_OPERATOR_CLUSTER")
	} else {
		Cluster = getClusterVersion()
	}

	if _, ok := Conf.Sidecars[Cluster]; !ok {
		glog.Errorf("Invalid config - section %s is missing. Fallback to default", Cluster)
		Cluster = "default"
		if _, ok := Conf.Sidecars[Cluster]; !ok {
			glog.Fatalf("Invalid config - section %s is missing", Cluster)
		}
	} else {
		glog.Infof("Using config section %s", Cluster)
	}
}
