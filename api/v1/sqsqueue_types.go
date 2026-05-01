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

// SQSQueueSpec defines the desired state of SQSQueue.
type SQSQueueSpec struct {
	// QueueName is the name of the SQS queue. FIFO queues must end with ".fifo".
	QueueName string `json:"queueName"`
	// Region is the AWS region where the queue will be created.
	Region string `json:"region"`
	// Fifo creates a FIFO queue when true. The QueueName must end with ".fifo".
	// +optional
	Fifo bool `json:"fifo,omitempty"`
	// VisibilityTimeoutSeconds is the duration (in seconds) that a message is
	// hidden after being received. Defaults to 30.
	// +optional
	VisibilityTimeoutSeconds int32 `json:"visibilityTimeoutSeconds,omitempty"`
	// MessageRetentionSeconds is how long (in seconds) messages are kept in the
	// queue before being deleted. Defaults to 345600 (4 days).
	// +optional
	MessageRetentionSeconds int32 `json:"messageRetentionSeconds,omitempty"`
	// Tags are key-value pairs applied to the SQS queue.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// SQSQueueStatus defines the observed state of SQSQueue.
type SQSQueueStatus struct {
	// QueueURL is the URL of the provisioned SQS queue.
	QueueURL string `json:"queueUrl,omitempty"`
	// QueueARN is the ARN of the provisioned SQS queue.
	QueueARN string `json:"queueArn,omitempty"`
	// State is the current lifecycle state of the queue (Active, Unknown).
	State string `json:"state,omitempty"`
}

// CreatedQueueInfo holds the details of a newly provisioned SQS queue.
type CreatedQueueInfo struct {
	QueueURL string
	QueueARN string
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="QueueName",type="string",JSONPath=".spec.queueName",description="The SQS queue name"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="The current state of the SQS queue"
// +kubebuilder:printcolumn:name="QueueURL",type="string",JSONPath=".status.queueUrl",description="The SQS queue URL"

// SQSQueue is the Schema for the sqsqueues API.
type SQSQueue struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of SQSQueue
	// +required
	Spec SQSQueueSpec `json:"spec"`

	// status defines the observed state of SQSQueue
	// +optional
	Status SQSQueueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SQSQueueList contains a list of SQSQueue.
type SQSQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQSQueue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SQSQueue{}, &SQSQueueList{})
}
