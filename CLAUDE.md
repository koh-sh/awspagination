# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`awspagination` is a Go static analysis linter that detects missing pagination handling in AWS SDK for Go v2 List API calls. It uses the `go/analysis` framework and type-based analysis to work automatically with any AWS service.

## Development Commands

### Testing
```bash
# Run all tests
make test

# Coverage analysis
make cov                              # Generate HTML coverage report and run octocov
go tool cover -func=coverage.out      # View coverage by function (run after make cov)
```

### Building
```bash
# Build standalone tool
make build

# Run on target code
./awspagination ./...
./awspagination ./path/to/project/...

# Run with configuration options
./awspagination -custom-fields=MyToken,CustomNextToken ./...
./awspagination -include-tests ./...  # Analyze test files
```

### Dependencies
```bash
# Update dependencies
make tidy

# Verify dependencies
go mod verify
```

## Configuration Options

The linter supports configuration via command-line flags:

### Custom Token Fields (`-custom-fields`)

Add custom pagination token field names in addition to the default fields.

**Usage**:
```bash
./awspagination -custom-fields=MyToken,CustomNextToken ./...
```

**Default fields** (always checked):
- `NextToken`, `NextMarker`, `Marker`, `NextContinuationToken`, `ContinuationToken`, `NextPageToken`, `NextPageMarker`

**Use case**: Your project uses custom response types with non-standard pagination field names.

### Include Test Files (`-include-tests`)

Analyze test files (`*_test.go`) in addition to regular source files.

**Usage**:
```bash
./awspagination -include-tests ./...
```

**Default**: `false` (test files are excluded from analysis)

**Use case**: Detect missing pagination handling in test helper functions or test code that makes real API calls.

## Code Architecture

### Core Components

**awspagination.go** - Main analyzer implementation
- `Config` struct: Configuration for custom token fields and test file inclusion
- `Analyzer` variable: Entry point for the go/analysis framework
- `apiSpecificPaginationFields`: Maps service names to their special pagination fields (DynamoDB, API Gateway)
- `init()`: Registers command-line flags for configuration
- `getPaginationTokenFields()`: Returns all token fields (default + custom)
- `run()`: Main analysis logic using `inspector.Nodes` for efficient AST traversal (~2.5x faster than `ast.Inspect`)
- `checkAssignment()`: Examines individual assignment statements for API calls
- `hasPaginationTokenField()`: Checks for both standard and service-specific pagination token fields
- `hasSpecificField()`: Helper function to check for service-specific fields regardless of type
- `hasPaginationTokenFieldRecursive()`: Recursively checks for standard pagination token fields
- `hasPaginationHandling()`: Detects pagination patterns (manual loops or Paginator usage)
- `isAWSSDKType()`: Validates that types originate from AWS SDK v2 to prevent false positives
- `isAWSSDKPackage()`: Checks if package is from AWS SDK v2
- `extractServiceNameFromPackage()`: Extracts service name from package path
- `extractAPICallInfo()`: Extracts service name, method name, and type information from API calls
- `buildErrorMessage()`: Constructs detailed, actionable error messages with code examples

**cmd/awspagination/main.go** - Standalone CLI tool
- Wraps the analyzer with `singlechecker.Main()` for command-line usage

### Analysis Flow

1. **AST Traversal**: Uses `inspector.Nodes` with filter for `*ast.FuncDecl` and `*ast.AssignStmt` nodes
2. **Function Context Tracking**: Maintains `currentFunc` to scope analysis to individual functions
3. **Type Checking**: Uses `pass.TypesInfo.Types` to get type information for call expressions
4. **Pagination Token Detection**: Recursively searches type hierarchy for pagination token fields
5. **AWS SDK Validation**: Verifies type package path contains `aws-sdk-go-v2/service/`
6. **Pattern Recognition**: Scans function body for:
   - Manual loop pattern: Access to `result.NextToken` (or other token fields)
   - Paginator pattern: Usage of `NewXXXPaginator`, `HasMorePages()`, or `NextPage()`

### Type System Details

The analyzer uses Go's `go/types` package extensively:
- **Pointer unwrapping**: Handles `*ResponseType` and `ResponseType` transparently
- **Named type resolution**: Unwraps named types to access underlying struct
- **Embedded field recursion**: Traverses embedded structs to find pagination tokens
- **Tuple handling**: Extracts first type from multiple return values `(result, error)`
- **Circular reference prevention**: Uses `seen` maps to prevent infinite recursion

### Supported Pagination Token Fields

The linter detects these pagination token fields:

**Standard fields** (checked for all services):
1. `NextToken` - Most common (100+ services)
2. `NextMarker` - EFS, ELB, ELBv2, KMS, Lambda, Route53, CloudFront
3. `Marker` - IAM, RDS, DMS, ElastiCache, Neptune, Redshift
4. `NextContinuationToken` - S3 ListObjectsV2
5. `ContinuationToken` - S3 ListObjectsV2 (echoed from input)
6. `NextPageToken` - CostExplorer, ServiceCatalog
7. `NextPageMarker` - Route53Domains

**Service-specific fields** (defined in `apiSpecificPaginationFields`):
- `LastEvaluatedKey` - DynamoDB (Query, Scan, etc.) - type: `map[string]types.AttributeValue`
- `Position` - API Gateway (GetRestApis, GetResources, etc.) - type: `*string`
- `IsTruncated`, `NextRecordName`, `NextRecordType`, `NextRecordIdentifier` - Route53 (ListResourceRecordSets) - multi-field pagination

**Adding new service-specific fields**:
1. Add entry to `apiSpecificPaginationFields` map with service name as key
2. Add test cases in `testdata/src/test/<service>.go`
3. Update dependencies in `testdata/src/test/go.mod`
4. Run `go mod vendor` in testdata/src/test
5. Update README.md documentation

## Testing Strategy

### Test Data Structure

Tests use `golang.org/x/tools/go/analysis/analysistest` framework with test packages:
- **testdata/src/test/**: Main test cases, organized by file:
  - **basic.go**: Main test cases (ECS, S3, IAM) with vendored AWS SDK v2 types
  - **dynamodb.go**: DynamoDB LastEvaluatedKey pagination tests
  - **apigateway.go**: API Gateway Position pagination tests
  - **route53.go**: Route53 multi-field pagination tests (IsTruncated, NextRecordName, etc.)
  - **embedded.go**: Tests for embedded struct patterns (wrapped responses)
  - **interface.go**: Tests for interface/wrapper patterns (testability patterns)
  - **nonawssdk.go**: Tests that non-AWS SDK types are not detected (false positive prevention)
- **testdata/src/testskip/**: Test file exclusion feature tests:
  - **basic.go**: Normal file with proper pagination handling
  - **skip_test.go**: Test file with missing pagination (for `-include-tests` flag testing)

### Test Pattern

The test package contains:
- Multiple Go source files, all with `package test`
- `// want "message"` comments to mark expected diagnostics
- Vendored AWS SDK dependencies to ensure type information is available
- Test cases covering both positive (should warn) and negative (should not warn) scenarios
- Tests that custom types with pagination-like fields (e.g., NextToken) are correctly ignored when AWS SDK is present

### Adding New Test Cases

1. Choose appropriate file in `testdata/src/test/` (basic.go, embedded.go, interface.go, or nonawssdk.go)
2. Add test case with `// want "missing pagination handling..."` comment where warning is expected
3. Ensure function names don't conflict with existing functions
4. Run `make test` to verify
5. Update coverage: `make cov`

## golangci-lint Integration

Key requirements for golangci-lint integration:
- Requires `LoadModeTypesInfo` for type checking
- Exports `Analyzer` variable for integration
- Uses `github.com/koh-sh/awspagination` import path
- Configuration via `Analyzer.Flags` (already implemented)

## Linter Scope and Limitations

**Same-Function Scope**: Following patterns from linters like `errcheck` and `bodyclose`, this linter only checks for pagination handling within the same function where the API call is made.

**Does NOT detect**:
- Pagination handling in separate helper functions
- Wrapper libraries that handle pagination internally
- Cross-function pagination patterns

For these cases, use `//nolint:awspagination` with a comment explaining why it's safe.

**Known Pagination Patterns Not Currently Detected** (intentionally out of scope):
- **Kinesis GetRecords**: Uses `NextShardIterator` - stream reading, not list pagination
- **CloudWatch Logs**: Uses `NextForwardToken`/`NextBackwardToken` - bidirectional streaming

**Now Supported via Service-Specific Fields**:
- **DynamoDB**: `LastEvaluatedKey` (type: `map[string]types.AttributeValue`) - Now detected
- **API Gateway**: `Position` (*string) - Now detected
- **Route53**: `IsTruncated`, `NextRecordName`, `NextRecordType`, `NextRecordIdentifier` - Multi-field pagination now detected

## Error Messages

**Design Philosophy**: Error messages are designed to be concise yet informative, explaining both the problem and its impact. Following best practices from linters like `errcheck` and `bodyclose`, messages are kept to 2-3 lines.

**Message Structure** (2-line format):

1. **Problem Description**: Clear identification of the issue with context
   - Example: `missing pagination handling for AWS SDK List API call (result has NextToken field)`

2. **Impact + Solution**: Explains why it's a problem and how to fix it
   - Example: `When there are many results, only the first page is returned. Use NewListTasksPaginator or loop with result.NextToken.`

**Complete Message Examples**:

```
missing pagination handling for AWS SDK List API call (result has NextToken field)
When there are many results, only the first page is returned. Use NewListTasksPaginator or loop with result.NextToken.
```

```
missing pagination handling for AWS SDK List API call (result has NextContinuationToken field)
When there are many results, only the first page is returned. Use NewListObjectsV2Paginator or loop with output.NextContinuationToken.
```

**Context Awareness**: Messages adapt based on available information:
- When service + method detected: Shows actual paginator name (e.g., `NewListObjectsV2Paginator`)
- When variable name known: Shows specific field access (e.g., `output.NextContinuationToken`)
- When information incomplete: Shows generic guidance (e.g., `Use a paginator or loop with NextMarker`)

**Testing**: Error message formatting is tested in `message_test.go` with coverage for various scenarios.

## Performance Considerations

- Uses `inspector.Nodes` instead of `ast.Inspect` for ~2.5x faster AST traversal
- Filters to only `FuncDecl` and `AssignStmt` nodes to minimize work
- Recursion prevention with `seen` maps to avoid infinite loops on circular types
- Type-based analysis means no need to maintain a list of AWS services
