package embercsi

import (
	"fmt"
	"strings" 
	"strconv" 
	embercsiv1alpha1 "github.com/embercsi/ember-csi-operator/pkg/apis/ember-csi/v1alpha1"
        logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// Default values
const (
        // Node DaemonSet's ServiceAccount
        NodeSA string           = "ember-csi-operator"
        // Controller StatefulSet's ServiceAccount
        ControllerSA string     = "ember-csi-operator"
	DEFAULT_CSI_SPEC 	= 0.3
)

// Global variables
var Conf 	*Config
var Cluster 	string
var CSI_SPEC	float64
var PluginDomainName	string

// validate Input from CR. Proceed if correct or else quit with a log
// This fn must be called after reading the config file and/or defaults set
func Validate(ecsi *embercsiv1alpha1.EmberCSI) {
        log := logf.Log.WithName("Validate")

	// Remove 'v' prefix if it exists
	if strings.HasPrefix(Conf.Sidecars[Cluster].CSISpecVersion, "v") {	// starts with 'v' e.g. v0.3
		var tmpConf = Conf.Sidecars[Cluster]
		tmpConf.CSISpecVersion = strings.Replace(Conf.Sidecars[Cluster].CSISpecVersion, "v", "", -1)
		Conf.Sidecars[Cluster] = tmpConf

		// Store CSI Spec version for future use
		spec, err := strconv.ParseFloat(tmpConf.CSISpecVersion, 64)
		CSI_SPEC = spec
		if err != nil {
			log.Info(fmt.Sprintf("Could't convert X_CSI_SPEC_VERSION to float: %d"), err.Error())
			// Use our sane default
			CSI_SPEC = DEFAULT_CSI_SPEC
		} 
	}

	// Plugin's domain name to use. Prior to CSI spec 1.0, we used reverse
	// domain name, after 1.0 we use forward.
	if CSI_SPEC >= 1.0 {
		PluginDomainName = fmt.Sprintf("%s.%s", ecsi.Name, "--provisioner=ember-csi.io")
	} else {
		PluginDomainName = fmt.Sprintf("%s.%s", "--provisioner=io.ember-csi", ecsi.Name)
	}
}
