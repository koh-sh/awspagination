package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/route53"
)

// Bad: No pagination handling for Route53
func badRoute53() {
	client := &route53.Client{}
	ctx := context.Background()
	input := &route53.ListResourceRecordSetsInput{}
	result, _ := client.ListResourceRecordSets(ctx, input) // want "missing pagination handling for AWS SDK List API call"
	_ = result
}

// Good: Manual loop with IsTruncated check (recommended pattern)
func goodRoute53IsTruncated() {
	client := &route53.Client{}
	ctx := context.Background()
	input := &route53.ListResourceRecordSetsInput{}
	for {
		result, err := client.ListResourceRecordSets(ctx, input)
		if err != nil {
			break
		}
		for _, rr := range result.ResourceRecordSets {
			_ = rr
		}
		// Check IsTruncated field
		if !result.IsTruncated {
			break
		}
		input.StartRecordName = result.NextRecordName
		input.StartRecordType = result.NextRecordType
		input.StartRecordIdentifier = result.NextRecordIdentifier
	}
}

// Good: Manual loop with NextRecordName check
func goodRoute53NextRecordName() {
	client := &route53.Client{}
	ctx := context.Background()
	input := &route53.ListResourceRecordSetsInput{}
	for {
		result, err := client.ListResourceRecordSets(ctx, input)
		if err != nil {
			break
		}
		for _, rr := range result.ResourceRecordSets {
			_ = rr
		}
		// Check NextRecordName field
		if result.NextRecordName == nil {
			break
		}
		input.StartRecordName = result.NextRecordName
		input.StartRecordType = result.NextRecordType
		input.StartRecordIdentifier = result.NextRecordIdentifier
	}
}

// Good: Manual loop with NextRecordType check
func goodRoute53NextRecordType() {
	client := &route53.Client{}
	ctx := context.Background()
	input := &route53.ListResourceRecordSetsInput{}
	for {
		result, err := client.ListResourceRecordSets(ctx, input)
		if err != nil {
			break
		}
		for _, rr := range result.ResourceRecordSets {
			_ = rr
		}
		// Check NextRecordType field
		if result.NextRecordType == "" {
			break
		}
		input.StartRecordName = result.NextRecordName
		input.StartRecordType = result.NextRecordType
	}
}

// Good: Using Paginator
func goodRoute53Paginator() {
	client := &route53.Client{}
	ctx := context.Background()
	input := &route53.ListResourceRecordSetsInput{}
	paginator := route53.NewListResourceRecordSetsPaginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			break
		}
		for _, rr := range page.ResourceRecordSets {
			_ = rr
		}
	}
}
