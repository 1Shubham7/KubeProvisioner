package controller

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	computev1 "github.com/1Shubham7/operator/api/v1"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func sqsClient(region string) *sqs.Client {
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		fmt.Println("Error loading AWS config for SQS:", err)
		os.Exit(1)
	}
	return sqs.NewFromConfig(cfg)
}

func createSQSQueue(ctx context.Context, queue *computev1.SQSQueue) (*computev1.CreatedQueueInfo, error) {
	l := log.FromContext(ctx)
	l.Info("Creating SQS queue", "queueName", queue.Spec.QueueName, "region", queue.Spec.Region)

	sqsc := sqsClient(queue.Spec.Region)

	attrs := map[string]string{}
	if queue.Spec.VisibilityTimeoutSeconds > 0 {
		attrs[string(sqstypes.QueueAttributeNameVisibilityTimeout)] = strconv.Itoa(int(queue.Spec.VisibilityTimeoutSeconds))
	}
	if queue.Spec.MessageRetentionSeconds > 0 {
		attrs[string(sqstypes.QueueAttributeNameMessageRetentionPeriod)] = strconv.Itoa(int(queue.Spec.MessageRetentionSeconds))
	}
	if queue.Spec.Fifo {
		attrs[string(sqstypes.QueueAttributeNameFifoQueue)] = "true"
	}

	input := &sqs.CreateQueueInput{
		QueueName:  aws.String(queue.Spec.QueueName),
		Attributes: attrs,
		Tags:       queue.Spec.Tags,
	}

	result, err := sqsc.CreateQueue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQS queue %q: %w", queue.Spec.QueueName, err)
	}
	l.Info("SQS queue created", "queueUrl", aws.ToString(result.QueueUrl))

	// Fetch the ARN from queue attributes.
	attrResult, err := sqsc.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       result.QueueUrl,
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ARN for SQS queue %q: %w", queue.Spec.QueueName, err)
	}

	return &computev1.CreatedQueueInfo{
		QueueURL: aws.ToString(result.QueueUrl),
		QueueARN: attrResult.Attributes[string(sqstypes.QueueAttributeNameQueueArn)],
	}, nil
}

func deleteSQSQueue(ctx context.Context, queueURL string, region string) error {
	l := log.FromContext(ctx)
	l.Info("Deleting SQS queue", "queueUrl", queueURL)

	sqsc := sqsClient(region)
	_, err := sqsc.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NonExistentQueue") {
			l.Info("SQS queue already gone", "queueUrl", queueURL)
			return nil
		}
		return fmt.Errorf("failed to delete SQS queue %q: %w", queueURL, err)
	}

	l.Info("SQS queue deleted", "queueUrl", queueURL)
	return nil
}

// checkSQSQueueExists returns the queue URL if the queue exists, empty string if not.
func checkSQSQueueExists(ctx context.Context, queueName string, region string) (bool, error) {
	sqsc := sqsClient(region)
	_, err := sqsc.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NonExistentQueue") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check SQS queue %q: %w", queueName, err)
	}
	return true, nil
}
