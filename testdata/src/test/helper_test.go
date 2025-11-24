package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// HelperWithoutPagination is a test helper that should be skipped by default
// No want comment - this should NOT trigger a warning when tests are run normally
func HelperWithoutPagination(ecsClient *ecs.Client) {
	ctx := context.Background()
	result, _ := ecsClient.ListTasks(ctx, &ecs.ListTasksInput{})
	_ = result
}
