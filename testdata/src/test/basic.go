package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Test cases using real AWS SDK types

// Bad: No pagination handling
func bad1() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}
	result, _ := client.ListTasks(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result
}

// Bad: Assign to variable but don't use NextToken
func bad2() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}
	result, err := client.ListTasks(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	if err != nil {
		return
	}
	// Using result but not NextToken
	for _, item := range result.TaskArns {
		_ = item
	}
}

// Good: Manual loop with NextToken
func good1() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}
	for {
		result, err := client.ListTasks(ctx, input)
		if err != nil {
			break
		}
		for _, item := range result.TaskArns {
			_ = item
		}
		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}
}

// Good: Using Paginator
func good2() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}
	paginator := ecs.NewListTasksPaginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			break
		}
		for _, item := range page.TaskArns {
			_ = item
		}
	}
}

// Good: API without NextToken (no pagination needed)
func good3() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.DescribeTasksInput{}
	result, _ := client.DescribeTasks(ctx, input)
	_ = result
}

// Good: Blank identifier (explicitly ignored)
func good4() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}
	_, _ = client.ListTasks(ctx, input)
}

// Bad: Multiple calls in same function
func bad3() {
	client := &ecs.Client{}
	ctx := context.Background()
	input1 := &ecs.ListTasksInput{}
	result1, _ := client.ListTasks(ctx, input1) // want "missing pagination handling for AWS SDK List API call"
	_ = result1

	input2 := &ecs.ListTasksInput{}
	result2, _ := client.ListTasks(ctx, input2) // want "missing pagination handling for AWS SDK List API call"
	_ = result2
}

// Good: Access NextToken field (even without proper loop)
func good5() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}
	result, _ := client.ListTasks(ctx, input)
	// Accessing NextToken field means user is aware of pagination
	if result.NextToken != nil {
		_ = result.NextToken
	}
}

// Test cases for other pagination token fields

// Bad: NextMarker not handled (S3 ListObjects)
func badNextMarker() {
	client := &s3.Client{}
	ctx := context.Background()
	input := &s3.ListObjectsInput{}
	result, _ := client.ListObjects(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result
}

// Good: NextMarker handled with manual loop
func goodNextMarker() {
	client := &s3.Client{}
	ctx := context.Background()
	input := &s3.ListObjectsInput{}
	for {
		result, err := client.ListObjects(ctx, input)
		if err != nil {
			break
		}
		for _, item := range result.Contents {
			_ = item
		}
		if result.NextMarker == nil {
			break
		}
		input.Marker = result.NextMarker
	}
}

// Bad: NextContinuationToken not handled (S3 ListObjectsV2)
func badNextContinuationToken() {
	client := &s3.Client{}
	ctx := context.Background()
	input := &s3.ListObjectsV2Input{}
	result, _ := client.ListObjectsV2(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result
}

// Good: NextContinuationToken handled
func goodNextContinuationToken() {
	client := &s3.Client{}
	ctx := context.Background()
	input := &s3.ListObjectsV2Input{}
	for {
		result, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			break
		}
		for _, item := range result.Contents {
			_ = item
		}
		if result.NextContinuationToken == nil {
			break
		}
		input.ContinuationToken = result.NextContinuationToken
	}
}

// Bad: Marker not handled (IAM ListUsers)
func badMarker() {
	client := &iam.Client{}
	ctx := context.Background()
	input := &iam.ListUsersInput{}
	result, _ := client.ListUsers(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result
}

// Good: Marker handled
func goodMarker() {
	client := &iam.Client{}
	ctx := context.Background()
	input := &iam.ListUsersInput{}
	for {
		result, err := client.ListUsers(ctx, input)
		if err != nil {
			break
		}
		for _, item := range result.Users {
			_ = item
		}
		if result.Marker == nil {
			break
		}
		input.Marker = result.Marker
	}
}

// Edge case tests

// Bad: API call in defer (pagination handling needed but deferred)
func badDefer() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}

	defer func() {
		result, _ := client.ListTasks(ctx, input) // want "missing pagination handling for AWS SDK List API call"
		_ = result
	}()
}

// Bad: Multiple API calls without pagination
func badMultipleCalls() {
	client := &ecs.Client{}
	ctx := context.Background()

	result1, _ := client.ListTasks(ctx, &ecs.ListTasksInput{}) // want "missing pagination handling for AWS SDK List API call"
	_ = result1

	result2, _ := client.ListTasks(ctx, &ecs.ListTasksInput{}) // want "missing pagination handling for AWS SDK List API call"
	_ = result2
}

// Good: Nested loop with proper pagination
func goodNestedLoop() {
	client := &ecs.Client{}
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		input := &ecs.ListTasksInput{}
		for {
			result, err := client.ListTasks(ctx, input)
			if err != nil {
				break
			}
			for _, task := range result.TaskArns {
				_ = task
			}
			if result.NextToken == nil {
				break
			}
			input.NextToken = result.NextToken
		}
	}
}

// Bad: Nested loop without pagination
func badNestedLoop() {
	client := &ecs.Client{}
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		result, _ := client.ListTasks(ctx, &ecs.ListTasksInput{}) // want "missing pagination handling for AWS SDK List API call"
		_ = result
	}
}

// Good: Result stored in struct field with pagination handling
func goodStructField() {
	type Holder struct {
		result *ecs.ListTasksOutput
	}

	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}
	holder := &Holder{}

	for {
		result, err := client.ListTasks(ctx, input)
		if err != nil {
			break
		}
		holder.result = result

		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}
}

// Bad: Assignment to underscore for error but not result
func badUnderscoreError() {
	client := &ecs.Client{}
	ctx := context.Background()
	input := &ecs.ListTasksInput{}

	result, _ := client.ListTasks(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result
}
