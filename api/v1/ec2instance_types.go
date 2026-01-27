/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Ec2InstanceSpec defines the desired state of Ec2Instance
type Ec2InstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	AmiID             string            `json:"amiID,omitempty"`
	SSHKey            string            `json:"sshkey,omitempty"`
	InstanceType      string            `json:"instanceType,omitempty"`
	SubNet            string            `json:"subnet"`
	Tags              map[string]string `json:"tags"`
	Storage           StorageConfig     `json:"storage"`
	AdditionalStorage []StorageConfig   `json:"additionalStorage,omitempty"`
}

type StorageConfig struct {
	VolumeSize int    `json:"volumeSize"`
	VolumeType string `json:"volumeType"`
}

// Ec2InstanceStatus defines the observed state of Ec2Instance.
type Ec2InstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Phase      string `json:"phase,omitempty"`
	InstanceID string `json:"instanceID,omitempty"`
	PublicIP   string `json:"publicIP,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Ec2Instance is the Schema for the ec2instances API
type Ec2Instance struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Ec2Instance
	// +required
	Spec Ec2InstanceSpec `json:"spec"`

	// status defines the observed state of Ec2Instance
	// +optional
	Status Ec2InstanceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// Ec2InstanceList contains a list of Ec2Instance
type Ec2InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Ec2Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ec2Instance{}, &Ec2InstanceList{})
}
