package controller

import (
	"fmt"
	"net/http"
	"os"
)

// AWSCredentialsChecker is a readiness checker that verifies AWS credentials
// are present. The operator cannot reconcile Ec2Instance resources without them.
func AWSCredentialsChecker(req *http.Request) error {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID is not set")
	}
	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return fmt.Errorf("AWS_SECRET_ACCESS_KEY is not set")
	}
	return nil
}
