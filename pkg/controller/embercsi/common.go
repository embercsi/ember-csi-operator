package embercsi

const (
        // Node DaemonSet's ServiceAccount
        NodeSA string           = "ember-csi-operator"
        // Controller StatefulSet's ServiceAccount
        ControllerSA string     = "ember-csi-operator"

        // Image Versions
        RegistrarVersion string   = "v0.3.0"
        AttacherVersion string    = "v0.3.0"
        ProvisionerVersion string = "v0.3.0"
        DriverVersion string      = "master"
)
