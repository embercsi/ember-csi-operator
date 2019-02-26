package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmberCSISpec defines the desired state of EmberCSI
type EmberCSISpec struct {
        Config          EmberCSIConfig    `json:"config"`
        Image           string            `json:"image",omitempty`
        NodeSelector	map[string]string `json:"nodeSelector",omitempty`
        Tolerations	[]v1.Toleration   `json:"tolerations",omitempty`
}

type Tolerations struct {
        Key        	string `json:"key,omitempty"`
        Operator	string `json:"operator,omitempty"`
        Value		string `json:"value,omitempty"`
        Effect		string `json:"effect,omitempty"`
}

// EmberCSIStatus defines the observed state of EmberCSI
type EmberCSIStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EmberCSI is the Schema for the embercsis API
// +k8s:openapi-gen=true
type EmberCSI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmberCSISpec   `json:"spec,omitempty"`
	Status EmberCSIStatus `json:"status,omitempty"`
}

type EmberCSIConfig struct {
        EnvVars         EnvVars `json:"envVars"`
        SysFiles        Secrets `json:"sysFiles"`
}

type EnvVars struct {
        X_CSI_BACKEND_CONFIG     string `json:"X_CSI_BACKEND_CONFIG",omitempty`
        X_CSI_EMBER_CONFIG       string `json:"X_CSI_EMBER_CONFIG",omitempty`
        X_CSI_PERSISTENCE_CONFIG string `json:"X_CSI_PERSISTENCE_CONFIG",omitempty`
        X_CSI_DEBUG_MODE         string `json:"X_CSI_DEBUG_MODE",omitempty`
        X_CSI_ABORT_DUPLICATES   string `json:"X_CSI_ABORT_DUPLICATES",omitempty`
        X_CSI_DEFAULT_MOUNT_FS   string `json:"X_CSI_DEFAULT_MOUNT_FS",omitempty`
        X_CSI_NODE_ID            string `json:"X_CSI_NODE_ID",omitempty`
        X_CSI_STORAGE_NW_IP      string `json:"X_CSI_STORAGE_NW_IP",omitempty`
        CSI_ENDPOINT             string `json:"CSI_ENDPOINT",omitempty`
        CSI_MODE                 string `json:"CSI_MODE",omitempty`
        EnvSecrets               []Secrets `json:"secret",omitempty`
} 

type Secrets struct {
        Name string `json:"name",omitempty`
        Key string `json:"key",omitempty`
} 

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EmberCSIList contains a list of EmberCSI
type EmberCSIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmberCSI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmberCSI{}, &EmberCSIList{})
}
