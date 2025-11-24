package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
)

// Test cases for API Gateway Position pagination

// Bad: No pagination handling for GetRestApis
func badAPIGatewayGetRestApis() {
	client := &apigateway.Client{}
	ctx := context.Background()
	input := &apigateway.GetRestApisInput{}
	result, _ := client.GetRestApis(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result.Items
}

// Bad: No pagination handling for GetResources
func badAPIGatewayGetResources() {
	client := &apigateway.Client{}
	ctx := context.Background()
	input := &apigateway.GetResourcesInput{}
	result, _ := client.GetResources(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result.Items
}

// Bad: No pagination handling for GetAuthorizers
func badAPIGatewayGetAuthorizers() {
	client := &apigateway.Client{}
	ctx := context.Background()
	input := &apigateway.GetAuthorizersInput{}
	result, _ := client.GetAuthorizers(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result.Items
}

// Good: Manual loop with Position for GetRestApis
func goodAPIGatewayGetRestApis() {
	client := &apigateway.Client{}
	ctx := context.Background()
	input := &apigateway.GetRestApisInput{}
	for {
		result, err := client.GetRestApis(ctx, input)
		if err != nil {
			break
		}
		_ = result.Items
		if result.Position == nil {
			break
		}
		input.Position = result.Position
	}
}

// Good: Manual loop with Position for GetResources
func goodAPIGatewayGetResources() {
	client := &apigateway.Client{}
	ctx := context.Background()
	input := &apigateway.GetResourcesInput{}
	for {
		result, err := client.GetResources(ctx, input)
		if err != nil {
			break
		}
		_ = result.Items
		if result.Position == nil {
			break
		}
		input.Position = result.Position
	}
}

// Good: Manual loop with Position for GetAuthorizers
func goodAPIGatewayGetAuthorizers() {
	client := &apigateway.Client{}
	ctx := context.Background()
	input := &apigateway.GetAuthorizersInput{}
	for {
		result, err := client.GetAuthorizers(ctx, input)
		if err != nil {
			break
		}
		_ = result.Items
		if result.Position == nil {
			break
		}
		input.Position = result.Position
	}
}
