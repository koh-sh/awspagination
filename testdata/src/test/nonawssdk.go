package test

import "context"

// This package tests that non-AWS SDK types are NOT detected
// even if they have pagination-like field names

// Custom type that happens to have NextToken field
// This should NOT be detected because it's not from AWS SDK
type CustomResult struct {
	Items     []string
	NextToken *string // This is NOT from AWS SDK
}

type CustomClient struct{}

func (c *CustomClient) ListItems(ctx context.Context) (*CustomResult, error) {
	return &CustomResult{}, nil
}

// This should NOT trigger the linter because CustomResult is not from AWS SDK
func testNonAWSSDK() {
	client := &CustomClient{}
	ctx := context.Background()
	result, _ := client.ListItems(ctx)
	_ = result
}

// Even with multiple calls, should not trigger
func testMultipleNonAWS() {
	client := &CustomClient{}
	ctx := context.Background()
	result1, _ := client.ListItems(ctx)
	_ = result1
	result2, _ := client.ListItems(ctx)
	_ = result2
}
