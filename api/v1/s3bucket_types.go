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

// S3BucketSpec defines the desired state of S3Bucket.
type S3BucketSpec struct {
	// BucketName is the globally unique name for the S3 bucket.
	BucketName string `json:"bucketName"`
	// Region is the AWS region where the bucket will be created.
	Region string `json:"region"`
	// Versioning enables S3 versioning on the bucket.
	// +optional
	Versioning bool `json:"versioning,omitempty"`
	// Tags are key-value pairs applied to the S3 bucket.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// S3BucketStatus defines the observed state of S3Bucket.
type S3BucketStatus struct {
	// BucketARN is the ARN of the provisioned S3 bucket.
	BucketARN string `json:"bucketArn,omitempty"`
	// Endpoint is the HTTPS endpoint of the bucket.
	Endpoint string `json:"endpoint,omitempty"`
	// State is the current lifecycle state of the bucket (Active, Unknown).
	State string `json:"state,omitempty"`
}

// CreatedBucketInfo holds the details of a newly provisioned S3 bucket.
type CreatedBucketInfo struct {
	BucketARN string
	Endpoint  string
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="BucketName",type="string",JSONPath=".spec.bucketName",description="The S3 bucket name"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="The current state of the S3 bucket"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".status.endpoint",description="The S3 bucket endpoint"

// S3Bucket is the Schema for the s3buckets API.
type S3Bucket struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of S3Bucket
	// +required
	Spec S3BucketSpec `json:"spec"`

	// status defines the observed state of S3Bucket
	// +optional
	Status S3BucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// S3BucketList contains a list of S3Bucket.
type S3BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []S3Bucket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&S3Bucket{}, &S3BucketList{})
}
