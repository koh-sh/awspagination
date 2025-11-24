package testskip

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// FunctionInTestFile demonstrates missing pagination in a test file
// This should be skipped by default, but detected with -include-tests=true
func FunctionInTestFile(ecsClient *ecs.Client) {
	ctx := context.Background()
	result, err := ecsClient.ListTasks(ctx, &ecs.ListTasksInput{}) // want "missing pagination handling for AWS SDK List API call"
	if err != nil {
		return
	}
	_ = result
}
