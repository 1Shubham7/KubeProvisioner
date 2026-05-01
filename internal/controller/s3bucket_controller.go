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

const s3BucketFinalizer = "s3bucket.compute.cloud.com"

// S3BucketReconciler reconciles an S3Bucket object.
type S3BucketReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=compute.cloud.com,resources=s3buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.cloud.com,resources=s3buckets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.cloud.com,resources=s3buckets/finalizers,verbs=update

func (r *S3BucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	s3Bucket := &computev1.S3Bucket{}
	if err := r.Get(ctx, req.NamespacedName, s3Bucket); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Deletion path: terminate the real bucket then drop the finalizer.
	if !s3Bucket.DeletionTimestamp.IsZero() {
		logger.Info("S3Bucket is being deleted", "bucketName", s3Bucket.Spec.BucketName)
		if err := deleteS3Bucket(ctx, s3Bucket.Spec.BucketName, s3Bucket.Spec.Region); err != nil {
			logger.Error(err, "Failed to delete S3 bucket — bucket may not be empty")
			return ctrl.Result{Requeue: true}, err
		}
		controllerutil.RemoveFinalizer(s3Bucket, s3BucketFinalizer)
		if err := r.Update(ctx, s3Bucket); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// Already provisioned: verify the real bucket still exists.
	if s3Bucket.Status.BucketARN != "" {
		logger.Info("S3Bucket already provisioned, verifying", "bucketName", s3Bucket.Spec.BucketName)
		exists, err := checkS3BucketExists(ctx, s3Bucket.Spec.BucketName, s3Bucket.Spec.Region)
		if err != nil {
			s3Bucket.Status.State = "Unknown"
			_ = r.Status().Update(ctx, s3Bucket)
			return ctrl.Result{Requeue: true}, err
		}
		if !exists {
			logger.Info("S3 bucket not found in AWS, clearing status", "bucketName", s3Bucket.Spec.BucketName)
			s3Bucket.Status.State = "Unknown"
			s3Bucket.Status.BucketARN = ""
			s3Bucket.Status.Endpoint = ""
			return ctrl.Result{Requeue: true}, r.Status().Update(ctx, s3Bucket)
		}
		return ctrl.Result{}, nil
	}

	// First time: add finalizer, then create the bucket.
	logger.Info("Creating S3 bucket", "bucketName", s3Bucket.Spec.BucketName)
	controllerutil.AddFinalizer(s3Bucket, s3BucketFinalizer)
	if err := r.Update(ctx, s3Bucket); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	info, err := createS3Bucket(ctx, s3Bucket)
	if err != nil {
		logger.Error(err, "Failed to create S3 bucket")
		return ctrl.Result{}, err
	}

	s3Bucket.Status.BucketARN = info.BucketARN
	s3Bucket.Status.Endpoint = info.Endpoint
	s3Bucket.Status.State = "Active"
	if err := r.Status().Update(ctx, s3Bucket); err != nil {
		logger.Error(err, "Failed to update S3Bucket status")
		return ctrl.Result{}, err
	}

	logger.Info("S3Bucket reconciled successfully", "bucketName", s3Bucket.Spec.BucketName, "arn", info.BucketARN)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *S3BucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.S3Bucket{}).
		Named("s3bucket").
		Complete(r)
}
