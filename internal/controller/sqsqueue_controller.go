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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	computev1 "github.com/1Shubham7/operator/api/v1"
)

const sqsQueueFinalizer = "sqsqueue.compute.cloud.com"

// SQSQueueReconciler reconciles an SQSQueue object.
type SQSQueueReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=compute.cloud.com,resources=sqsqueues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.cloud.com,resources=sqsqueues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.cloud.com,resources=sqsqueues/finalizers,verbs=update

func (r *SQSQueueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	sqsQueue := &computev1.SQSQueue{}
	if err := r.Get(ctx, req.NamespacedName, sqsQueue); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Deletion path: delete the real queue then drop the finalizer.
	if !sqsQueue.DeletionTimestamp.IsZero() {
		logger.Info("SQSQueue is being deleted", "queueName", sqsQueue.Spec.QueueName)
		if sqsQueue.Status.QueueURL != "" {
			if err := deleteSQSQueue(ctx, sqsQueue.Status.QueueURL, sqsQueue.Spec.Region); err != nil {
				logger.Error(err, "Failed to delete SQS queue")
				return ctrl.Result{Requeue: true}, err
			}
		}
		controllerutil.RemoveFinalizer(sqsQueue, sqsQueueFinalizer)
		if err := r.Update(ctx, sqsQueue); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// Already provisioned: verify the real queue still exists.
	if sqsQueue.Status.QueueURL != "" {
		logger.Info("SQSQueue already provisioned, verifying", "queueName", sqsQueue.Spec.QueueName)
		exists, err := checkSQSQueueExists(ctx, sqsQueue.Spec.QueueName, sqsQueue.Spec.Region)
		if err != nil {
			sqsQueue.Status.State = "Unknown"
			_ = r.Status().Update(ctx, sqsQueue)
			return ctrl.Result{Requeue: true}, err
		}
		if !exists {
			logger.Info("SQS queue not found in AWS, clearing status", "queueName", sqsQueue.Spec.QueueName)
			sqsQueue.Status.State = "Unknown"
			sqsQueue.Status.QueueURL = ""
			sqsQueue.Status.QueueARN = ""
			return ctrl.Result{Requeue: true}, r.Status().Update(ctx, sqsQueue)
		}
		return ctrl.Result{}, nil
	}

	// First time: add finalizer, then create the queue.
	logger.Info("Creating SQS queue", "queueName", sqsQueue.Spec.QueueName)
	controllerutil.AddFinalizer(sqsQueue, sqsQueueFinalizer)
	if err := r.Update(ctx, sqsQueue); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	info, err := createSQSQueue(ctx, sqsQueue)
	if err != nil {
		logger.Error(err, "Failed to create SQS queue")
		return ctrl.Result{}, err
	}

	sqsQueue.Status.QueueURL = info.QueueURL
	sqsQueue.Status.QueueARN = info.QueueARN
	sqsQueue.Status.State = "Active"
	if err := r.Status().Update(ctx, sqsQueue); err != nil {
		logger.Error(err, "Failed to update SQSQueue status")
		return ctrl.Result{}, err
	}

	logger.Info("SQSQueue reconciled successfully",
		"queueName", sqsQueue.Spec.QueueName,
		"queueUrl", info.QueueURL)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SQSQueueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.SQSQueue{}).
		Named("sqsqueue").
		Complete(r)
}
