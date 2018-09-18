package stub

import (
	"strings"
        "gopkg.in/yaml.v2"
        "io/ioutil"
	"github.com/sirupsen/logrus"
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
		logrus.Infof("Cannot Open Config File. Will use defaults.\n")
                return DefaultConfig()
        }
        err = yaml.Unmarshal(source, &Conf)
        if err != nil {
		logrus.Infof("Cannot Open Config File. Will use defaults.\n")
		return DefaultConfig()
        }

        return Conf 
}

// Populate the Config Stuct with some default values and Return it
func DefaultConfig () *Config {
	driver := map[string]string {
		"default":"akrog/ember-csi:master",
	}
	Conf.Cluster = "ocp"
	Conf.Images.Attacher = "quay.io/k8scsi/csi-attacher:v0.3.0"
	Conf.Images.Provisioner = "quay.io/k8scsi/csi-provisioner:v0.3.0"
	Conf.Images.Registrar = "quay.io/k8scsi/driver-registrar:v0.3.0"
	Conf.Images.Driver = driver
	return Conf
}
