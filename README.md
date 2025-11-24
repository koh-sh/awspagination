# awspagination

A golangci-lint linter that detects missing pagination handling in AWS SDK v2 List API calls.

## Overview

AWS SDK for Go v2 List APIs have default result limits and require pagination handling using `NextToken`. This linter automatically detects missing pagination implementations to prevent bugs caused by incomplete data retrieval.

This linter is designed for integration with golangci-lint, but can also be used as a standalone command-line tool.

## How It Works

This linter detects missing pagination handling by analyzing your code in three steps:

### 1. Identifies paginated API calls

Verifies the API call is from AWS SDK v2 by checking the package path (`github.com/aws/aws-sdk-go-v2/service/...`).

Then checks if the response type has pagination token fields:

- **Standard fields** (checked for all services): `NextToken`, `NextMarker`, `Marker`, `NextContinuationToken`, `ContinuationToken`, `NextPageToken`, `NextPageMarker`
- **Service-specific fields**: `LastEvaluatedKey` (DynamoDB), `Position` (API Gateway), `IsTruncated`/`NextRecordName`/`NextRecordType`/`NextRecordIdentifier` (Route53)

See [Detected Pagination Token Fields](#detected-pagination-token-fields) for the complete list with service details.

Uses Go's type system to automatically work with any AWS service without maintaining a service list.

### 2. Looks for pagination handling patterns

Within the **same function**, searches for either:

- **Manual loop**: Accesses pagination token field (e.g., `result.NextToken`)
- **Paginator**: Uses `NewXXXPaginator`, `HasMorePages()`, or `NextPage()`

### 3. Reports if pagination is missing

If a paginated API call is found without any pagination handling pattern in the same function, a warning is reported.

**Important**: This linter only checks within the same function scope. If you handle pagination in a separate helper function or wrapper library, use `//nolint:awspagination` to suppress the warning.

## Installation & Configuration

### With golangci-lint

There are two integration methods available:

#### Method 1: Direct Integration (Recommended)

Add to your `.golangci.yml`:

```yaml
linters-settings:
  custom:
    awspagination:
      path: github.com/koh-sh/awspagination
      description: Detects missing pagination handling in AWS SDK v2 List API calls
      original-url: https://github.com/koh-sh/awspagination
  awspagination:
    # Add custom pagination token field names (optional)
    custom-fields:
      - MyToken
      - CustomNextToken
    # Include test files in analysis (optional, default: false)
    include-tests: false

linters:
  enable:
    - awspagination
```

#### Method 2: Module Plugin

Add to your `.golangci.yml`:

```yaml
linters-settings:
  custom:
    awspagination:
      path: github.com/koh-sh/awspagination
      type: "module"
      description: Detects missing pagination handling in AWS SDK v2 List API calls
      original-url: https://github.com/koh-sh/awspagination
      settings:
        custom-fields: ["MyToken", "CustomNextToken"]
        include-tests: true

linters:
  enable:
    - awspagination
```

**Note:** Both methods provide the same functionality. Method 1 (direct integration) is recommended for simplicity. Method 2 (module plugin) is useful if you need advanced plugin features.

### As a standalone tool

```bash
# Installation
go install github.com/koh-sh/awspagination/cmd/awspagination@latest

# Basic usage
awspagination ./...

# With custom pagination token fields
awspagination -custom-fields=MyToken,CustomNextToken ./...

# Include test files
awspagination -include-tests ./...
```

## Configuration Options

### Custom Token Fields

Add custom pagination token field names in addition to the default fields.

**Use case**: Your project uses custom response types with non-standard pagination field names.

**Default fields**: See [Detected Pagination Token Fields](#detected-pagination-token-fields) for the complete list.

### Include Test Files

Analyze test files (`*_test.go`) in addition to regular source files.

**Default**: `false` (test files are excluded from analysis)

**Use case**: Detect missing pagination handling in test helper functions or test code that makes real API calls.

**golangci-lint configuration**:

```yaml
linters-settings:
  awspagination:
    include-tests: true
```

## Examples

### ❌ Bad: No pagination handling

```go
func bad() {
    client := ecs.NewFromConfig(cfg)
    result, _ := client.ListTasks(ctx, &ecs.ListTasksInput{})
    // Warning: missing pagination handling for AWS SDK List API call
    for _, task := range result.TaskArns {
        fmt.Println(task)
    }
}
```

### ✅ Good: Manual loop with NextToken

```go
func good1() {
    client := ecs.NewFromConfig(cfg)
    input := &ecs.ListTasksInput{}

    for {
        result, err := client.ListTasks(ctx, input)
        if err != nil {
            break
        }

        for _, task := range result.TaskArns {
            fmt.Println(task)
        }

        if result.NextToken == nil {
            break
        }
        input.NextToken = result.NextToken
    }
}
```

### ✅ Good: Using Paginator

```go
func good2() {
    client := ecs.NewFromConfig(cfg)
    input := &ecs.ListTasksInput{}

    paginator := ecs.NewListTasksPaginator(client, input)
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            break
        }

        for _, task := range page.TaskArns {
            fmt.Println(task)
        }
    }
}
```

### ✅ Good: Intentionally limited (using nolint)

```go
func good3() {
    client := ecs.NewFromConfig(cfg)
    input := &ecs.ListTasksInput{
        MaxResults: aws.Int32(10),
    }

    //nolint:awspagination // Only need first 10 results
    result, _ := client.ListTasks(ctx, input)
    for _, task := range result.TaskArns {
        fmt.Println(task)
    }
}
```

## Supported

- **SDK Version**: AWS SDK for Go v2 only (`github.com/aws/aws-sdk-go-v2`)
- **Test Files**: Excluded by default (use `-include-tests` flag to include)

### Detected Pagination Token Fields

| Field Name | Scope | Services/Usage |
|------------|-------|----------------|
| `NextToken` | All Services | Most common - ECS, EC2, Lambda, etc. (100+ services) |
| `NextMarker` | All Services | S3 ListObjects, EFS, ELB, ELBv2, KMS, Lambda, Route53, CloudFront |
| `Marker` | All Services | IAM, RDS, DMS, ElastiCache, Neptune, Redshift |
| `NextContinuationToken` | All Services | S3 ListObjectsV2 |
| `ContinuationToken` | All Services | S3 ListObjectsV2 (input echo) |
| `NextPageToken` | All Services | CostExplorer, ServiceCatalog |
| `NextPageMarker` | All Services | Route53Domains |
| `LastEvaluatedKey` | DynamoDB | Query, Scan, etc. |
| `Position` | API Gateway | GetRestApis, GetResources, etc. |
| `IsTruncated` / `NextRecordName` / `NextRecordType` / `NextRecordIdentifier` | Route53 | ListResourceRecordSets - any field indicates pagination |

**All Services** fields are checked for all AWS services. **DynamoDB**, **API Gateway**, and **Route53** fields are only checked for their respective services.

## Development

### Run tests

```bash
make test
```

### Build

```bash
make build
```

### Test on your code

```bash
./awspagination ./your-project/...
```
