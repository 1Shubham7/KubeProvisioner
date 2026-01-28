package controller

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	computev1 "github.com/1Shubham7/operator/api/v1"
	"github.com/1Shubham7/operator/utils/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func awsClient(region string) *ec2.Client {
	// read env variable for namespace
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")))
	if err != nil {
		fmt.Println("Error loading AWS config:", err)
		os.Exit(1)
	}
	return ec2.NewFromConfig(cfg)
}

func checkEC2InstanceExists(ctx context.Context, instanceID string, ec2Instance *computev1.Ec2Instance) (bool, *ec2types.Instance, error) {
	// create the client for ec2 instance
	fmt.Println("Checking instance ", instanceID)
	ec2Client := awsClient(ec2Instance.Spec.Region)

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	}

	result, err := ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
			return false, nil, nil
		}
		return false, nil, err
	}

	fmt.Println("Legnth of Reservations are ", len(result.Reservations))

	// Check if we got any instances back
	if len(result.Reservations) == 0 {
		// No reservations means the instance is not found or not running
		return false, nil, nil
	}
	return true, &result.Reservations[0].Instances[0], nil
}


func createEc2Instance(ec2Instance *computev1.Ec2Instance) (createdInstanceInfo *computev1.CreatedInstanceInfo, err error) {
	l := log.Log.WithName("createEc2Instance")

	l.Info("=== STARTING EC2 INSTANCE CREATION ===",
		"ami", ec2Instance.Spec.AMIId,
		"instanceType", ec2Instance.Spec.InstanceType,
		"region", ec2Instance.Spec.Region)

	// create the client for ec2 instance
	ec2Client := awsClient(ec2Instance.Spec.Region)

	// create the input for the run instances
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Instance.Spec.AMIId),
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		KeyName:      aws.String(ec2Instance.Spec.KeyPair),
		SubnetId:     aws.String(ec2Instance.Spec.Subnet),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		//SecurityGroupIds: []string{ec2Instance.Spec.SecurityGroups[0]},
	}

	l.Info("=== CALLING AWS RunInstances API ===")
	// run the instances
	result, err := ec2Client.RunInstances(context.TODO(), runInput)
	if err != nil {
		l.Error(err, "Failed to create EC2 instance")
		return nil, fmt.Errorf("failed to create EC2 instance: %w", err)
	}

	if len(result.Instances) == 0 {
		l.Error(nil, "No instances returned in RunInstancesOutput")
		fmt.Println("No instances returned in RunInstancesOutput")
		return nil, nil
	}

	// Till here, the instance is created and we have
	// Instance ID, private dns and IP, instance type and image id.
	inst := result.Instances[0]
	l.Info("=== EC2 INSTANCE CREATED SUCCESSFULLY ===", "instanceID", *inst.InstanceId)

	// Now we need to wait for the instance to be running and then get the public ip and dns
	l.Info("=== WAITING FOR INSTANCE TO BE RUNNING ===")

	runWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
	maxWaitTime := 3 * time.Minute // Increased from 10 seconds - instances typically take 30-60 seconds

	err = runWaiter.Wait(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}, maxWaitTime)
	if err != nil {
		l.Error(err, "Failed to wait for instance to be running")
		return nil, fmt.Errorf("failed to wait for instance to be running: %w", err)
	}

	// After creating the instance, we waited and now we describe to
	// 1. Get the public IP and dns as it takes some time for it
	// 2. Getting the state of the instance.
	// We do this so we can send the instance's state to the status of the custom resource. for user to see with kubectl get ec2instances
	l.Info("=== CALLING AWS DescribeInstances API TO GET INSTANCE DETAILS ===")
	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}

	describeResult, err := ec2Client.DescribeInstances(context.TODO(), describeInput)
	if err != nil {
		l.Error(err, "Failed to describe EC2 instance")
		return nil, fmt.Errorf("failed to describe EC2 instance: %w", err)
	}

	fmt.Println("Describe result", "public ip", *describeResult.Reservations[0].Instances[0].PublicDnsName, "state", describeResult.Reservations[0].Instances[0].State.Name)

	// You get "invalid memory address or nil pointer dereference" here if any of the following are true:
	// - result.Instances is nil or has length 0
	// - Any of the pointer fields (e.g., PublicIpAddress, PrivateIpAddress, etc.) are nil

	// To avoid this, always check for nil and length before dereferencing:

	// Wait for a bit to allow instance fields to be populated

	fmt.Printf("Private IP of the instance: %v", helpers.DerefString(inst.PrivateIpAddress))
	fmt.Printf("State of the instance: %v", describeResult.Reservations[0].Instances[0].State.Name)
	fmt.Printf("Private DNS of the instance: %v", helpers.DerefString(inst.PrivateDnsName))
	fmt.Printf("Instance ID of the instance: %v", helpers.DerefString(inst.InstanceId))
	fmt.Println("Instance Type of the instance: ", inst.InstanceType)
	fmt.Printf("Image ID of the instance: %v", helpers.DerefString(inst.ImageId))
	fmt.Printf("Key Name of the instance: %v", helpers.DerefString(inst.KeyName))

	// block until the instance is running
	// blockUntilInstanceRunning(ctx, ec2Instance.Status.InstanceID, ec2Instance)

	// Get the instance details safely (public IP/DNS might be nil for private subnets)
	instance := describeResult.Reservations[0].Instances[0]
	createdInstanceInfo = &computev1.CreatedInstanceInfo{
		InstanceID: *inst.InstanceId,
		State:      string(instance.State.Name),
		PublicIP:   helpers.DerefString(instance.PublicIpAddress),
		PrivateIP:  helpers.DerefString(instance.PrivateIpAddress),
		PublicDNS:  helpers.DerefString(instance.PublicDnsName),
		PrivateDNS: helpers.DerefString(instance.PrivateDnsName),
	}

	l.Info("=== EC2 INSTANCE CREATION COMPLETED ===",
		"instanceID", createdInstanceInfo.InstanceID,
		"state", createdInstanceInfo.State,
		"publicIP", createdInstanceInfo.PublicIP)

	// Optionally, update ec2Instance.Status.InstanceID = *result.Instances[0].InstanceId

	// For now, just return nil to indicate success.
	return createdInstanceInfo, nil
}

func deleteEc2Instance(ctx context.Context, ec2Instance *computev1.Ec2Instance) (bool, error) {
	l := log.FromContext(ctx)

	l.Info("Deleting EC2 instance", "instanceID", ec2Instance.Status.InstanceID)

	// create the client for ec2 instance
	ec2Client := awsClient(ec2Instance.Spec.Region)

	// Terminate the instance
	terminateResult, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	})

	if err != nil {
		l.Error(err, "Failed to terminate EC2 instance")
		return false, err
	}

	l.Info("Instance termination initiated",
		"instanceID", ec2Instance.Status.InstanceID,
		"currentState", terminateResult.TerminatingInstances[0].CurrentState.Name)

	// Use the AWS SDK v2 waiter to efficiently wait for instance termination
	// The waiter uses exponential backoff and is more efficient than manual polling
	waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)

	// Configure waiter options
	maxWaitTime := 5 * time.Minute // Maximum time to wait for termination
	waitParams := &ec2.DescribeInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	}

	l.Info("Waiting for instance to be terminated",
		"instanceID", ec2Instance.Status.InstanceID,
		"maxWaitTime", maxWaitTime)

	// Wait for the instance to be terminated
	err = waiter.Wait(ctx, waitParams, maxWaitTime)

	if err != nil {
		l.Error(err, "Failed while waiting for instance termination",
			"instanceID", ec2Instance.Status.InstanceID)
		return false, err
	}

	l.Info("EC2 instance successfully terminated", "instanceID", ec2Instance.Status.InstanceID)
	return true, nil
}
