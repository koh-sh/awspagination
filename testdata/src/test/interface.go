package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// Test cases for interface-based patterns
// Common pattern: code uses interfaces for testability

// Interface for ECS client operations (testability pattern)
type ECSClient interface {
	ListTasks(ctx context.Context, input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error)
}

// Service layer that uses interface
type Service struct {
	client ECSClient // interface instead of concrete type
}

func NewService(client ECSClient) *Service {
	return &Service{client: client}
}

// Bad: Interface method call without pagination handling
func (s *Service) testInterfaceMethod() {
	ctx := context.Background()

	// s.client is an interface (ECSClient)
	// but the return type is *ecs.ListTasksOutput which has NextToken
	result, _ := s.client.ListTasks(ctx, &ecs.ListTasksInput{}) // want "missing pagination handling"

	_ = result
}

// Good: Interface method call with proper pagination
func (s *Service) testInterfaceMethodGood() {
	ctx := context.Background()
	input := &ecs.ListTasksInput{}

	for {
		result, err := s.client.ListTasks(ctx, input)
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

// Bad: Assigning AWS SDK response to interface{}
func testInterfaceReturn() {
	client := &ecs.Client{}
	ctx := context.Background()

	var result interface{}
	result, _ = client.ListTasks(ctx, &ecs.ListTasksInput{}) // want "missing pagination handling"
	_ = result
}

// Bad: Type assertion but still no pagination handling
func testTypeAssertion() {
	client := &ecs.Client{}
	ctx := context.Background()

	var result interface{}
	result, _ = client.ListTasks(ctx, &ecs.ListTasksInput{}) // want "missing pagination handling"

	if typed, ok := result.(*ecs.ListTasksOutput); ok {
		_ = typed
	}
}

// Bad: Using 'any' type (Go 1.18+)
func testAny() {
	client := &ecs.Client{}
	ctx := context.Background()

	var result any
	result, _ = client.ListTasks(ctx, &ecs.ListTasksInput{}) // want "missing pagination handling"
	_ = result
}

// Good: Using type assertion with pagination handling
// Note: The linter currently cannot track pagination handling through type assertions
// This is a known limitation - consider avoiding interface{} for AWS SDK responses
func testTypeAssertionGood() {
	client := &ecs.Client{}
	ctx := context.Background()

	// Direct assignment (no interface{})
	result, _ := client.ListTasks(ctx, &ecs.ListTasksInput{})

	// Access NextToken field directly
	if result.NextToken != nil {
		_ = result.NextToken
	}
}
