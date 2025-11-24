package awspagination

import (
	"strings"
	"testing"
)

// TestErrorMessageFormat verifies that the error message contains all expected components
func TestErrorMessageFormat(t *testing.T) {
	tests := []struct {
		name        string
		tokenFields []string
		varName     string
		info        apiCallInfo
		wantParts   []string
	}{
		{
			name:        "complete information",
			tokenFields: []string{"NextToken"},
			varName:     "result",
			info: apiCallInfo{
				methodName:  "ListTasks",
				serviceName: "ecs",
				typeName:    "ListTasksOutput",
			},
			wantParts: []string{
				"missing pagination handling for AWS SDK List API call",
				"result has NextToken field",
				"When there are many results, only the first page is returned",
				"NewListTasksPaginator",
				"loop with",
				"result.NextToken",
			},
		},
		{
			name:        "S3 ListObjectsV2 with NextContinuationToken",
			tokenFields: []string{"NextContinuationToken"},
			varName:     "output",
			info: apiCallInfo{
				methodName:  "ListObjectsV2",
				serviceName: "s3",
				typeName:    "ListObjectsV2Output",
			},
			wantParts: []string{
				"missing pagination handling for AWS SDK List API call",
				"result has NextContinuationToken field",
				"When there are many results, only the first page is returned",
				"NewListObjectsV2Paginator",
				"loop with",
				"output.NextContinuationToken",
			},
		},
		{
			name:        "minimal information",
			tokenFields: []string{"NextMarker"},
			varName:     "",
			info:        apiCallInfo{},
			wantParts: []string{
				"missing pagination handling for AWS SDK List API call",
				"result has NextMarker field",
				"When there are many results, only the first page is returned",
				"a paginator",
				"loop with",
				"NextMarker",
			},
		},
		{
			name:        "Route53 multi-field pagination",
			tokenFields: []string{"IsTruncated", "NextRecordName", "NextRecordType", "NextRecordIdentifier"},
			varName:     "result",
			info: apiCallInfo{
				methodName:  "ListResourceRecordSets",
				serviceName: "route53",
				typeName:    "ListResourceRecordSetsOutput",
			},
			wantParts: []string{
				"missing pagination handling for AWS SDK List API call",
				"result has IsTruncated, NextRecordName, NextRecordType, and NextRecordIdentifier fields",
				"When there are many results, only the first page is returned",
				"NewListResourceRecordSetsPaginator",
				"loop with",
				"result.IsTruncated",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildErrorMessage(tt.tokenFields, tt.varName, tt.info)

			// Verify all expected parts are present
			for _, want := range tt.wantParts {
				if !strings.Contains(msg, want) {
					t.Errorf("buildErrorMessage() missing expected part %q\nGot:\n%s", want, msg)
				}
			}

			// Print the message for manual inspection
			t.Logf("Generated message:\n%s", msg)
		})
	}
}

// TestAPICallInfoExtraction verifies that we can extract service and method names correctly
func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		pkgPath     string
		wantService string
	}{
		{
			pkgPath:     "github.com/aws/aws-sdk-go-v2/service/ecs",
			wantService: "ecs",
		},
		{
			pkgPath:     "github.com/aws/aws-sdk-go-v2/service/s3",
			wantService: "s3",
		},
		{
			pkgPath:     "github.com/aws/aws-sdk-go-v2/service/dynamodb/types",
			wantService: "dynamodb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.pkgPath, func(t *testing.T) {
			got := extractServiceNameFromPackage(tt.pkgPath)
			if got != tt.wantService {
				t.Errorf("extractServiceNameFromPackage(%q) = %q, want %q", tt.pkgPath, got, tt.wantService)
			}
		})
	}
}
