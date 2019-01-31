package embercsi

import (
	"strings"
        "gopkg.in/yaml.v2"
        "io/ioutil"
        "encoding/json"
        logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
        "fmt"
)

// Global Var to Store Config
var Conf *Config

type Config struct {
        Cluster string `yaml:"cluster"`
        Images  struct {
                Attacher        string `yaml:"csi-attacher"`
                Provisioner     string `yaml:"csi-provisioner"`
                Registrar       string `yaml:"driver-registrar"`
                Driver          map[string]string `yaml:"ember-csi-driver"`
        } `yaml:"images"`
}

func (config *Config) getDriverImage( backend_config string, image string ) string {
	var backend_config_map map[string]string
	json.Unmarshal([]byte(backend_config), &backend_config_map)
	backend := backend_config_map["driver"]
	log := logf.Log.WithName("config")

	if len(image) == 0 {
		if len(backend) > 0 && len(config.Images.Driver[backend]) > 0 {
			image = config.Images.Driver[backend]
		} else if len(config.Images.Driver["default"]) > 0 {
			image = config.Images.Driver["default"]
		} else {
			image = "embercsi/ember-csi:master"
		}
	}
	log.Info(fmt.Sprintf("Using driver image %s", image))
	return image
}

func (config *Config) getCluster() string {
        return config.Cluster
}

func ReadConfig(configFile *string) {
	Conf = NewConfig(configFile)
}

// Config factory
func NewConfig ( configFile *string ) *Config {
	// If configFile is not specified. Lets use our default
	if len(strings.TrimSpace(*configFile)) == 0 {
		*configFile = "/etc/ember-csi-operator/config.yml"
	}

        source, err := ioutil.ReadFile(*configFile)
        if err != nil {
		//logrus.Infof("Cannot Open Config File. Will use defaults.\n")
                return DefaultConfig()
        }
        err = yaml.Unmarshal(source, &Conf)
        if err != nil {
		//logrus.Infof("Cannot Open Config File. Will use defaults.\n")
		return DefaultConfig()
        }

        return Conf 
}

// Populate the Config Stuct with some default values and Return it
func DefaultConfig () *Config {
	driver := map[string]string {
		"default":"embercsi/ember-csi:master",
	}
	Conf.Cluster = "ocp"
	Conf.Images.Attacher = "quay.io/k8scsi/csi-attacher:v0.3.0"
	Conf.Images.Provisioner = "quay.io/k8scsi/csi-provisioner:v0.3.0"
	Conf.Images.Registrar = "quay.io/k8scsi/driver-registrar:v0.3.0"
	Conf.Images.Driver = driver
	return Conf
}
