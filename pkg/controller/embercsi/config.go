package embercsi

import (
	"strings"
        "gopkg.in/yaml.v2"
        "io/ioutil"
)

// Global Var to Store Config
var Conf *Config

type Config struct {
        Cluster string `yaml:"cluster"`
        Images  struct {
                Attacher        string `yaml:"csi-attacher"`
                Provisioner     string `yaml:"csi-provisioner"`
                Registrar       string `yaml:"driver-registrar"`
                Driver          string `yaml:"ember-csi-driver"`
        } `yaml:"images"`
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
	Conf.Cluster = "ocp"
	Conf.Images.Attacher = "registry.redhat.io/openshift3/csi-attacher:v3.11"
	Conf.Images.Provisioner = "registry.redhat.io/openshift3/csi-provisioner:v3.11"
	Conf.Images.Registrar = "registry.redhat.io/openshift3/csi-driver-registrar:v3.11"
	Conf.Images.Driver = "akrog/ember-csi:master"
	return Conf
}
