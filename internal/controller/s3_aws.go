package controller

import (
	"context"
	"fmt"
	"os"
	"strings"

	computev1 "github.com/1Shubham7/operator/api/v1"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func s3Client(region string) *s3.Client {
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		fmt.Println("Error loading AWS config for S3:", err)
		os.Exit(1)
	}
	return s3.NewFromConfig(cfg)
}

func createS3Bucket(ctx context.Context, bucket *computev1.S3Bucket) (*computev1.CreatedBucketInfo, error) {
	l := log.FromContext(ctx)
	l.Info("Creating S3 bucket", "bucketName", bucket.Spec.BucketName, "region", bucket.Spec.Region)

	s3c := s3Client(bucket.Spec.Region)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket.Spec.BucketName),
	}
	// us-east-1 must NOT include a LocationConstraint; all other regions must.
	if bucket.Spec.Region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(bucket.Spec.Region),
		}
	}

	_, err := s3c.CreateBucket(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 bucket %q: %w", bucket.Spec.BucketName, err)
	}
	l.Info("S3 bucket created", "bucketName", bucket.Spec.BucketName)

	if len(bucket.Spec.Tags) > 0 {
		tagSet := make([]s3types.Tag, 0, len(bucket.Spec.Tags))
		for k, v := range bucket.Spec.Tags {
			tagSet = append(tagSet, s3types.Tag{Key: aws.String(k), Value: aws.String(v)})
		}
		_, err = s3c.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
			Bucket:  aws.String(bucket.Spec.BucketName),
			Tagging: &s3types.Tagging{TagSet: tagSet},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to tag S3 bucket %q: %w", bucket.Spec.BucketName, err)
		}
	}

	if bucket.Spec.Versioning {
		_, err = s3c.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
			Bucket: aws.String(bucket.Spec.BucketName),
			VersioningConfiguration: &s3types.VersioningConfiguration{
				Status: s3types.BucketVersioningStatusEnabled,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to enable versioning on S3 bucket %q: %w", bucket.Spec.BucketName, err)
		}
	}

	arn := fmt.Sprintf("arn:aws:s3:::%s", bucket.Spec.BucketName)
	endpoint := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket.Spec.BucketName, bucket.Spec.Region)

	return &computev1.CreatedBucketInfo{
		BucketARN: arn,
		Endpoint:  endpoint,
	}, nil
}

func deleteS3Bucket(ctx context.Context, bucketName string, region string) error {
	l := log.FromContext(ctx)
	l.Info("Deleting S3 bucket", "bucketName", bucketName)

	s3c := s3Client(region)

	_, err := s3c.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchBucket") {
			l.Info("S3 bucket already gone", "bucketName", bucketName)
			return nil
		}
		return fmt.Errorf("failed to delete S3 bucket %q: %w", bucketName, err)
	}

	l.Info("S3 bucket deleted", "bucketName", bucketName)
	return nil
}

func checkS3BucketExists(ctx context.Context, bucketName string, region string) (bool, error) {
	s3c := s3Client(region)

	result, err := s3c.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return false, fmt.Errorf("failed to list S3 buckets: %w", err)
	}
	for _, b := range result.Buckets {
		if aws.ToString(b.Name) == bucketName {
			return true, nil
		}
	}
	return false, nil
}
