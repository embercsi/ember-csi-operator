package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EmberCSIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []EmberCSI `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EmberCSI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              EmberCSISpec   `json:"spec"`
	Status            EmberCSIStatus `json:"status,omitempty"`
}

type EmberCSISpec struct {
	// Size is the size of the Controller StatefulSet
	Size 		int32  		`json:"size"`
	Config		EmberCSIConfig	`json:"config"`
}

type EmberCSIStatus struct {
	Phase string	`json:"phase"`
}


type EmberCSIConfig struct {
	EnvVars		EnvVars	`json:"envVars"`
	SysFiles	Secrets `json:"sysfiles"`
}

type EnvVars struct {
	X_CSI_BACKEND_CONFIG     string `json:"X_CSI_BACKEND_CONFIG",omitempty`
	X_CSI_EMBER_CONFIG     	 string `json:"X_CSI_EMBER_CONFIG",omitempty`
	X_CSI_PERSISTENCE_CONFIG string `json:"X_CSI_PERSISTENCE_CONFIG",omitempty`
	X_CSI_DEBUG_MODE	 string `json:"X_CSI_DEBUG_MODE",omitempty`
	X_CSI_ABORT_DUPLICATES	 string `json:"X_CSI_ABORT_DUPLICATES",omitempty`
	X_CSI_DEFAULT_MOUNT_FS	 string `json:"X_CSI_DEFAULT_MOUNT_FS",omitempty`
	X_CSI_NODE_ID	 	 string `json:"X_CSI_NODE_ID",omitempty`
	X_CSI_STORAGE_NW_IP	 string `json:"X_CSI_STORAGE_NW_IP",omitempty`
	CSI_ENDPOINT	 	 string `json:"CSI_ENDPOINT",omitempty`
	CSI_MODE	 	 string `json:"CSI_MODE",omitempty`
	EnvSecrets		 []Secrets `json:"secret",omitempty`
} 

type Secrets struct {
	Name string `json:"name",omitempty`
	Key string `json:"key",omitempty`
} 
