package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Test cases for DynamoDB LastEvaluatedKey pagination

// Bad: No pagination handling for Query
func badDynamoDBQuery() {
	client := &dynamodb.Client{}
	ctx := context.Background()
	input := &dynamodb.QueryInput{}
	result, _ := client.Query(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result.Items
}

// Bad: No pagination handling for Scan
func badDynamoDBScan() {
	client := &dynamodb.Client{}
	ctx := context.Background()
	input := &dynamodb.ScanInput{}
	result, _ := client.Scan(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result.Items
}

// Good: Using QueryPaginator
func goodDynamoDBQueryPaginator() {
	client := &dynamodb.Client{}
	ctx := context.Background()
	input := &dynamodb.QueryInput{}
	paginator := dynamodb.NewQueryPaginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			break
		}
		_ = page.Items
	}
}

// Good: Using ScanPaginator
func goodDynamoDBScanPaginator() {
	client := &dynamodb.Client{}
	ctx := context.Background()
	input := &dynamodb.ScanInput{}
	paginator := dynamodb.NewScanPaginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			break
		}
		_ = page.Items
	}
}

// Good: Manual loop with LastEvaluatedKey for Query
func goodDynamoDBQueryManual() {
	client := &dynamodb.Client{}
	ctx := context.Background()
	input := &dynamodb.QueryInput{}
	for {
		result, err := client.Query(ctx, input)
		if err != nil {
			break
		}
		_ = result.Items
		if result.LastEvaluatedKey == nil {
			break
		}
		input.ExclusiveStartKey = result.LastEvaluatedKey
	}
}

// Good: Manual loop with LastEvaluatedKey for Scan
func goodDynamoDBScanManual() {
	client := &dynamodb.Client{}
	ctx := context.Background()
	input := &dynamodb.ScanInput{}
	for {
		result, err := client.Scan(ctx, input)
		if err != nil {
			break
		}
		_ = result.Items
		if result.LastEvaluatedKey == nil {
			break
		}
		input.ExclusiveStartKey = result.LastEvaluatedKey
	}
}

// Good: Check length of LastEvaluatedKey
func goodDynamoDBQueryCheckLength() {
	client := &dynamodb.Client{}
	ctx := context.Background()
	input := &dynamodb.QueryInput{}
	for {
		result, err := client.Query(ctx, input)
		if err != nil {
			break
		}
		_ = result.Items
		if len(result.LastEvaluatedKey) == 0 {
			break
		}
		input.ExclusiveStartKey = result.LastEvaluatedKey
	}
}
