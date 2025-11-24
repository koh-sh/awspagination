package testskip

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// NormalFunctionWithPagination demonstrates a function with proper pagination handling
// This should NOT trigger a warning
func NormalFunctionWithPagination(ecsClient *ecs.Client) {
	ctx := context.Background()
	result, err := ecsClient.ListTasks(ctx, &ecs.ListTasksInput{})
	if err != nil {
		return
	}
	// Proper pagination handling
	_ = result.NextToken
}
