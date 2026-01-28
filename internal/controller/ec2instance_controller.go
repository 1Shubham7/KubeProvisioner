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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	computev1 "github.com/1Shubham7/operator/api/v1"
)

// Ec2InstanceReconciler reconciles a Ec2Instance object
type Ec2InstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ec2Instance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.0/pkg/reconcile
func (r *Ec2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	ec2Instance := &computev1.Ec2Instance{}
	if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("No instance found - stoping the reconcile loop")
			// Kubernetes will not retry
			return ctrl.Result{}, nil
		}
		// Kubernetes will retry with exponential backoff
		return ctrl.Result{}, err
	}

	//check if deletionTimestamp is not zero
	if !ec2Instance.DeletionTimestamp.IsZero() {
		logger.Info("Instance is being deleted, deleting the ec2 instance from aws as well")
		_, err := deleteEc2Instance(ctx, ec2Instance)
		if err != nil {
			logger.Error(err, "Failed to delete EC2 instance")
			// Kubernetes will retry with exponential backoff
			return ctrl.Result{Requeue: true}, err
		}

		// Remove the finalizer
		controllerutil.RemoveFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			logger.Error(err, "Failed to remove finalizer")
			// Kubernetes will retry with exponential backoff
			return ctrl.Result{Requeue: true}, err
		}
		// the instance state is terminated and the finalizer is removed
		return ctrl.Result{}, nil
	}

	// If there is no instance ID, that means instance was not created
	// This is to ensure we dont create two instances for same manifest
	if ec2Instance.Status.InstanceID != "" {
		logger.Info("Requested object already exists in Kubernetes. Not creating a new instance.", "instanceID", ec2Instance.Status.InstanceID)
		
		// Verify if the ec2 instance in aws is still running
		instanceExist, instanceState, err := checkEC2InstanceExists(ctx, ec2Instance.Status.InstanceID, ec2Instance)
		if err != nil {
			// Instance might be terminated, clear status and recreate
			ec2Instance.Status.InstanceID = ""
			ec2Instance.Status.State = ""
			ec2Instance.Status.PublicIP = ""
			ec2Instance.Status.PrivateIP = ""
			ec2Instance.Status.PublicDNS = ""
			ec2Instance.Status.PrivateDNS = ""
			return ctrl.Result{Requeue: true}, r.Status().Update(ctx, ec2Instance)
		}
		if !instanceExist {
			logger.Info("Instance does not exist or is not running", "instanceID", ec2Instance.Status.InstanceID)
			ec2Instance.Status.State = "Unknown"
			ec2Instance.Status.PublicIP = ""
			r.Status().Update(ctx, ec2Instance)
			return ctrl.Result{}, nil
		}
		if instanceExist && ec2Instance.Status.State == "Unknown" {
			logger.Info("Found a running Instance", "instanceID", ec2Instance.Status.InstanceID)
			ec2Instance.Status.State = string(instanceState.State.Name)
			ec2Instance.Status.PublicIP = *instanceState.PublicIpAddress
			r.Status().Update(ctx, ec2Instance)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	logger.Info("Creating new instance")

	ec2Instance.Finalizers = append(ec2Instance.Finalizers, "ec2instance.compute.cloud.com")
	if err := r.Update(ctx, ec2Instance); err != nil { // this will also trigger reconcile func
		logger.Error(err, "failed to add finalizer")
		return ctrl.Result{
			Requeue: true,
		}, err
	}

	createdInstanceInfo, err := createEc2Instance(ec2Instance)
	if err != nil {
		logger.Error(err, "Failed to create EC2 instance")
		// Kubernetes will retry with exponential backoff
		return ctrl.Result{}, err
	}

	ec2Instance.Status.InstanceID = createdInstanceInfo.InstanceID
	ec2Instance.Status.State = createdInstanceInfo.State
	ec2Instance.Status.PublicIP = createdInstanceInfo.PublicIP
	ec2Instance.Status.PrivateIP = createdInstanceInfo.PrivateIP
	ec2Instance.Status.PublicDNS = createdInstanceInfo.PublicDNS
	ec2Instance.Status.PrivateDNS = createdInstanceInfo.PrivateDNS

	err = r.Status().Update(ctx, ec2Instance)
	if err != nil {
		logger.Error(err, "Failed to update status")
		// Kubernetes will retry with backoff
		return ctrl.Result{}, err
	}
	logger.Info("Status updated - Reconcile loop will be triggered again")
	// Kubernetes will not retry - done, wait for next event
	return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Ec2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.Ec2Instance{}).
		Named("ec2instance").
		Complete(r)
}
