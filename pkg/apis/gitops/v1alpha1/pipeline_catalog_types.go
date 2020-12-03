package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PipelineCatalogFileName default name of the kpt strategy file
	PipelineCatalogFileName = "pipeline-catalog.yaml"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineCatalog represents a collection quickstart project
//
// +k8s:openapi-gen=true
type PipelineCatalog struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the PipelineCatalog from the client
	// +optional
	Spec PipelineCatalogSpec `json:"spec"`
}

// PipelineCatalogList contains a list of Repositories
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PipelineCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineCatalog `json:"items"`
}

// PipelineCatalogSpec defines the desired state of PipelineCatalog.
type PipelineCatalogSpec struct {
	// Repositories the repositories containing pipeline catalogs
	Repositories []PipelineCatalogSource `json:"repositories,omitempty"`
}

// PipelineCatalogSource the source of a pipeline catalog
type PipelineCatalogSource struct {
	ID     string `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	Label  string `json:"label,omitempty" protobuf:"bytes,2,opt,name=label"`
	GitURL string `json:"gitUrl,omitempty" protobuf:"bytes,3,opt,name=gitUrl"`
	GitRef string `json:"gitRef,omitempty" protobuf:"bytes,4,opt,name=gitRef"`
}
