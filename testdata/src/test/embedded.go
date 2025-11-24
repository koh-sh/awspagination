package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// Test cases for embedded fields
// AWS SDK responses are often wrapped in custom types

// Custom wrapper that embeds AWS SDK response
type WrappedResponse struct {
	*ecs.ListTasksOutput // embedded AWS SDK response
	RequestID            string
}

// Mock client that returns wrapped response
type Client struct{}

func (c *Client) GetTasks(ctx context.Context) (*WrappedResponse, error) {
	return &WrappedResponse{}, nil
}

func (c *Client) GetTasksDirect(ctx context.Context) (*ecs.ListTasksOutput, error) {
	return &ecs.ListTasksOutput{}, nil
}

// Bad: Direct AWS SDK type without pagination handling
func testDirect() {
	client := &Client{}
	ctx := context.Background()
	result, _ := client.GetTasksDirect(ctx) // want "missing pagination handling"

	_ = result
}

// Bad: Wrapped response without pagination handling
func testWrapped() {
	client := &Client{}
	ctx := context.Background()
	result, _ := client.GetTasks(ctx) // want "missing pagination handling"

	_ = result
}

// Good: Wrapped response with NextToken access
func testWrappedGood() {
	client := &Client{}
	ctx := context.Background()
	result, _ := client.GetTasks(ctx)

	// Access embedded NextToken field through promotion
	if result.NextToken != nil {
		_ = result.NextToken
	}
}
