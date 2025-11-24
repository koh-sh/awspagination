package awspagination

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `check for missing pagination handling in AWS SDK List API calls

This linter detects calls to AWS SDK v2 List APIs that return pagination tokens
(NextToken, NextMarker, NextContinuationToken, etc.) but don't implement pagination handling.`

// Default pagination token field names used across AWS services
var defaultPaginationTokenFields = []string{
	"NextToken",             // Most common (100+ services)
	"NextMarker",            // EFS, ELB, ELBv2, KMS, Lambda, Route53, CloudFront
	"Marker",                // IAM, RDS, DMS, ElastiCache, Neptune, Redshift
	"NextContinuationToken", // S3 ListObjectsV2
	"ContinuationToken",     // S3 ListObjectsV2 (echoed from input)
	"NextPageToken",         // CostExplorer, ServiceCatalog
	"NextPageMarker",        // Route53Domains
}

// apiSpecificPaginationFields maps AWS service names to their special pagination field names.
// These fields are checked in addition to the default pagination token fields.
// This map only includes services that use non-standard pagination fields.
//
// Key: service name (lowercase, e.g., "dynamodb", "apigateway")
// Value: list of pagination field names specific to that service
//
// To add support for a new service with special pagination fields:
// 1. Add an entry to this map with the service name and field names
// 2. Add test cases in testdata/src/test/<service>.go
// 3. Update README.md to document the new support
var apiSpecificPaginationFields = map[string][]string{
	"dynamodb":   {"LastEvaluatedKey"},                                                        // map[string]types.AttributeValue
	"apigateway": {"Position"},                                                                // *string
	"route53":    {"IsTruncated", "NextRecordName", "NextRecordType", "NextRecordIdentifier"}, // multi-field pagination
}

// Config holds the configuration for the analyzer.
// This type is exported to support future golangci-lint module plugin integration,
// where settings are decoded from YAML using mapstructure (which requires exported fields).
// For current direct integration via Analyzer.Flags, the export is not strictly necessary,
// but maintaining it provides forward compatibility.
type Config struct {
	// CustomTokenFields are additional pagination token field names to check.
	// These are added to the default fields, not replacing them.
	CustomTokenFields stringSliceFlag

	// IncludeTests determines whether to analyze test files (*_test.go).
	// Default is false (test files are excluded from analysis).
	IncludeTests bool
}

// stringSliceFlag implements flag.Value interface for comma-separated string slice flags.
// This type is exported because it's used as a field type in the exported Config struct.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	if value == "" {
		return nil
	}
	*s = append(*s, strings.Split(value, ",")...)
	return nil
}

// Settings holds the configuration for golangci-lint module plugin integration.
// This struct is used when the analyzer is loaded as a module plugin, where
// settings are decoded from YAML configuration files using mapstructure.
// For direct integration via Analyzer.Flags, use the Config struct instead.
type Settings struct {
	// CustomFields are additional pagination token field names to check.
	// These are added to the default fields, not replacing them.
	// Example YAML: custom-fields: ["MyToken", "CustomNextToken"]
	CustomFields []string `json:"custom-fields" mapstructure:"custom-fields"`

	// IncludeTests determines whether to analyze test files (*_test.go).
	// Default is false (test files are excluded from analysis).
	// Example YAML: include-tests: true
	IncludeTests bool `json:"include-tests" mapstructure:"include-tests"`
}

// config is the package-level configuration instance populated via command-line flags.
// Note: This variable is mutable and shared across analyzer invocations.
// In test environments with concurrent execution, tests should restore the original
// config in defer blocks to avoid side effects.
var config Config

// Analyzer is the awspagination analyzer.
// It can be used standalone or integrated into golangci-lint.
//
// For golangci-lint integration, this analyzer requires LoadModeTypesInfo
// because it uses pass.TypesInfo to check types.
var Analyzer = &analysis.Analyzer{
	Name:     "awspagination",
	Doc:      Doc,
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func init() {
	Analyzer.Flags.Var(&config.CustomTokenFields, "custom-fields",
		"comma-separated list of custom pagination token field names (in addition to default fields)")
	Analyzer.Flags.BoolVar(&config.IncludeTests, "include-tests", false,
		"analyze test files (*_test.go) in addition to regular source files (default: false)")
}

// New creates a new analyzer instance for golangci-lint module plugin integration.
// This function is called when the analyzer is loaded as a module plugin.
// It decodes settings from YAML configuration and applies them to the analyzer.
//
// For direct integration via Analyzer.Flags, this function is not used.
// The two integration methods are mutually exclusive:
// - Module plugin: Uses this New function with Settings from YAML
// - Direct integration: Uses Analyzer.Flags with command-line flags
//
// Example YAML configuration:
//
//	linters-settings:
//	  custom:
//	    awspagination:
//	      type: "module"
//	      settings:
//	        custom-fields: ["MyToken", "CustomNextToken"]
//	        include-tests: true
func New(settings any) ([]*analysis.Analyzer, error) {
	s, err := register.DecodeSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	// Apply settings to the package-level config
	// Convert []string to stringSliceFlag
	config.CustomTokenFields = stringSliceFlag(s.CustomFields)
	config.IncludeTests = s.IncludeTests

	return []*analysis.Analyzer{Analyzer}, nil
}

// getPaginationTokenFields returns all pagination token fields to check.
// Returns a new slice containing default fields plus any custom fields
// configured via the -custom-fields flag.
// The returned slice is a copy to prevent modification of the default field list.
func getPaginationTokenFields() []string {
	fields := make([]string, len(defaultPaginationTokenFields))
	copy(fields, defaultPaginationTokenFields)
	fields = append(fields, config.CustomTokenFields...)
	return fields
}

func run(pass *analysis.Pass) (any, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Use inspector.Nodes for efficient traversal with context tracking
	// This is more efficient than using ast.Inspect inside inspector.Preorder
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.AssignStmt)(nil),
	}

	var currentFunc *ast.FuncDecl

	// inspector.Nodes is more efficient than ast.Inspect (~2.5x faster).
	// The callback is invoked twice for each node: once when entering (push=true)
	// and once when exiting (push=false). This allows us to track context:
	// - When entering a FuncDecl, we save it to currentFunc
	// - When processing AssignStmt, we check it against currentFunc's body
	// - When exiting a FuncDecl, we reset currentFunc to nil
	inspector.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
		// Skip test files by default (unless -include-tests is specified)
		if !config.IncludeTests {
			pos := pass.Fset.Position(n.Pos())
			if strings.HasSuffix(pos.Filename, "_test.go") {
				return false // Skip this node and its children
			}
		}

		if push {
			// Entering a node: traveling down the AST tree
			switch node := n.(type) {
			case *ast.FuncDecl:
				// Track the current function scope for context
				currentFunc = node
			case *ast.AssignStmt:
				// Only process assignments inside functions (skip package-level assignments)
				if currentFunc == nil || currentFunc.Body == nil {
					return true
				}
				checkAssignment(pass, node, currentFunc)
			}
		} else {
			// Exiting a node: traveling back up the AST tree
			// Clear currentFunc when we exit a function declaration
			if _, ok := n.(*ast.FuncDecl); ok {
				currentFunc = nil
			}
		}
		return true
	})

	return nil, nil
}

// extractResultType extracts the result type from a call expression.
// Handles both single return values and tuple types (multiple return values).
// Returns the first type in case of multiple return values, or nil if extraction fails.
func extractResultType(pass *analysis.Pass, callExpr *ast.CallExpr) types.Type {
	typeAndValue, ok := pass.TypesInfo.Types[callExpr]
	if !ok {
		return nil
	}

	// Handle tuple types (multiple return values like (result, error))
	if tuple, ok := typeAndValue.Type.(*types.Tuple); ok {
		if tuple.Len() > 0 {
			return tuple.At(0).Type()
		}
		return nil
	}

	return typeAndValue.Type
}

// extractVariableName extracts the variable name from the left-hand side of an assignment.
// Returns empty string if the left-hand side is not a simple identifier or is the blank identifier "_".
func extractVariableName(lhs ast.Expr) string {
	ident, ok := lhs.(*ast.Ident)
	if !ok {
		return ""
	}
	// Skip blank identifier
	if ident.Name == "_" {
		return ""
	}
	return ident.Name
}

// checkAssignment checks a single assignment statement for missing pagination handling.
// It examines each call expression on the right-hand side and reports diagnostics
// for AWS SDK List API calls that lack proper pagination handling.
func checkAssignment(pass *analysis.Pass, assignStmt *ast.AssignStmt, funcDecl *ast.FuncDecl) {
	// Check each right-hand side expression
	for i, rightHandSide := range assignStmt.Rhs {
		callExpr, ok := rightHandSide.(*ast.CallExpr)
		if !ok {
			continue
		}

		// Get the corresponding left-hand side
		if i >= len(assignStmt.Lhs) {
			continue
		}

		// Extract result type from the call expression
		resultType := extractResultType(pass, callExpr)
		if resultType == nil {
			continue
		}

		// Extract API call information to get service name
		apiInfo := extractAPICallInfo(callExpr, resultType)

		// Check if the result type has a pagination token field
		// Pass service name to enable service-specific field detection
		tokenField := hasPaginationTokenField(resultType, apiInfo.serviceName)
		if tokenField == "" {
			continue
		}

		// Check if the type is from AWS SDK v2
		// This prevents false positives from non-AWS code
		if !isAWSSDKType(resultType) {
			continue
		}

		// Extract the variable name being assigned to
		varName := extractVariableName(assignStmt.Lhs[i])
		if varName == "" {
			continue
		}

		// Get all pagination token fields for this service
		// For multi-field pagination (e.g., Route53), we check if any field is accessed
		allTokenFields := getAllPaginationTokenFields(resultType, apiInfo.serviceName)
		if len(allTokenFields) == 0 {
			continue
		}

		// Check if pagination handling exists in the same function
		if hasPaginationHandling(funcDecl.Body, varName, allTokenFields) {
			continue
		}

		// Report the issue with detailed, actionable message
		pass.Report(analysis.Diagnostic{
			Pos:     callExpr.Pos(),
			Message: buildErrorMessage(allTokenFields, varName, apiInfo),
		})
	}
}

// getAllPaginationTokenFields returns all pagination token field names for a given type and service.
// This is used for services with multi-field pagination (e.g., Route53) where we need to check
// if any of the fields are accessed, not just the first one found.
// Returns a slice of field names that exist in the type.
func getAllPaginationTokenFields(t types.Type, serviceName string) []string {
	var fields []string

	// Check service-specific pagination fields if service is known
	if serviceName != "" {
		if serviceFields, ok := apiSpecificPaginationFields[strings.ToLower(serviceName)]; ok {
			for _, field := range serviceFields {
				// Create a new seen map for each field check to avoid false negatives
				seen := make(map[types.Type]bool)
				if hasSpecificField(t, field, seen) {
					fields = append(fields, field)
				}
			}
			// If we found service-specific fields, return them
			if len(fields) > 0 {
				return fields
			}
		}
	}

	// Check default pagination token fields
	seen := make(map[types.Type]bool)
	defaultField := hasPaginationTokenFieldRecursive(t, seen)
	if defaultField != "" {
		fields = append(fields, defaultField)
	}

	return fields
}

// hasPaginationTokenField checks if the type has any pagination token field.
// It checks both service-specific pagination fields and default pagination token fields.
// Returns the field name if found, empty string otherwise.
// This function checks both direct fields and embedded struct fields recursively.
func hasPaginationTokenField(t types.Type, serviceName string) string {
	seen := make(map[types.Type]bool)

	// First check service-specific pagination fields if service is known
	if serviceName != "" {
		if fields, ok := apiSpecificPaginationFields[strings.ToLower(serviceName)]; ok {
			for _, field := range fields {
				if hasSpecificField(t, field, seen) {
					return field
				}
			}
		}
	}

	// Then check default pagination token fields
	return hasPaginationTokenFieldRecursive(t, seen)
}

// hasSpecificField checks if a type has a specific field name.
// This is used for service-specific pagination fields that may have different types.
// Returns true if the field exists, regardless of its type.
func hasSpecificField(t types.Type, fieldName string, seen map[types.Type]bool) bool {
	// Unwrap pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Prevent infinite recursion
	if seen[t] {
		return false
	}
	seen[t] = true

	// Get underlying type from named types
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}

	// Check if it's a struct
	st, ok := t.(*types.Struct)
	if !ok {
		return false
	}

	// Look for the specific field
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field.Name() == fieldName {
			return true
		}
	}

	// Check embedded fields
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field.Embedded() {
			if hasSpecificField(field.Type(), fieldName, seen) {
				return true
			}
		}
	}

	return false
}

// hasPaginationTokenFieldRecursive recursively checks for pagination token fields
// The seen map prevents infinite recursion on circular struct embeddings
func hasPaginationTokenFieldRecursive(t types.Type, seen map[types.Type]bool) string {
	// Unwrap pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Prevent infinite recursion
	if seen[t] {
		return ""
	}
	seen[t] = true

	// Get underlying type from named types
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}

	// Check if it's a struct
	st, ok := t.(*types.Struct)
	if !ok {
		return ""
	}

	// Look for any pagination token field (in priority order)
	// Check direct fields first (prioritizing Next* fields over input fields)
	for _, tokenField := range getPaginationTokenFields() {
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			if field.Name() == tokenField {
				return tokenField
			}
		}
	}

	// Check embedded fields
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field.Embedded() {
			if tokenField := hasPaginationTokenFieldRecursive(field.Type(), seen); tokenField != "" {
				return tokenField
			}
		}
	}
	return ""
}

// hasPaginationHandling checks if pagination handling exists in the function body.
// It detects two patterns of pagination implementation:
//  1. Manual loop: Direct access to pagination token field (e.g., result.NextToken, result.NextMarker)
//     For multi-field pagination (e.g., Route53), checks if ANY of the fields are accessed
//  2. Paginator: Usage of AWS SDK paginator (NewXXXPaginator, HasMorePages, NextPage methods)
//
// Returns true if either pattern is found, indicating that pagination is properly handled.
func hasPaginationHandling(body *ast.BlockStmt, varName string, tokenFields []string) bool {
	// Pattern 1: Manual loop with pagination token access
	hasTokenAccess := false

	// Pattern 2: Paginator usage
	hasPaginatorUsage := false

	ast.Inspect(body, func(node ast.Node) bool {
		// Check for pagination token field access (e.g., result.NextToken, result.NextMarker)
		if sel, ok := node.(*ast.SelectorExpr); ok {
			// Check if accessing any of the pagination token fields
			for _, tokenField := range tokenFields {
				if sel.Sel.Name == tokenField {
					if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == varName {
						hasTokenAccess = true
						break
					}
				}
			}
		}

		// Check for Paginator usage
		if callExpr, ok := node.(*ast.CallExpr); ok {
			if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				// NewXXXPaginator pattern
				if len(sel.Sel.Name) > 9 && sel.Sel.Name[len(sel.Sel.Name)-9:] == "Paginator" {
					hasPaginatorUsage = true
				}
				// HasMorePages, NextPage methods
				if sel.Sel.Name == "HasMorePages" || sel.Sel.Name == "NextPage" {
					hasPaginatorUsage = true
				}
			}
		}

		return true
	})

	return hasTokenAccess || hasPaginatorUsage
}

// isAWSSDKType checks if the type originates from AWS SDK v2
// This function recursively checks the type and all embedded types to determine
// if any part of the type hierarchy comes from AWS SDK.
// This handles:
// - Direct AWS SDK types
// - Custom structs that embed AWS SDK types
// - Types from AWS SDK forks (as long as they maintain the service/ path structure)
// - Types from proxied/mirrored AWS SDK
func isAWSSDKType(t types.Type) bool {
	return isAWSSDKTypeRecursive(t, make(map[types.Type]bool))
}

// isAWSSDKTypeRecursive recursively checks if a type or any of its embedded types
// originates from AWS SDK v2
func isAWSSDKTypeRecursive(t types.Type, seen map[types.Type]bool) bool {
	// Unwrap pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Prevent infinite recursion on circular type references
	if seen[t] {
		return false
	}
	seen[t] = true

	// Check if this is a named type from AWS SDK
	if named, ok := t.(*types.Named); ok {
		pkg := named.Obj().Pkg()
		if pkg != nil && isAWSSDKPackage(pkg.Path()) {
			return true
		}
		// Check the underlying type (in case it's a type alias or has embedded fields)
		t = named.Underlying()
	}

	// For struct types, check all fields (including embedded fields)
	// This handles custom structs that embed AWS SDK types
	if st, ok := t.(*types.Struct); ok {
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			if isAWSSDKTypeRecursive(field.Type(), seen) {
				return true
			}
		}
	}

	return false
}

// isAWSSDKPackage checks if a package path is from AWS SDK v2.
// Uses Contains instead of HasPrefix to handle various SDK distribution scenarios:
// - Official SDK: github.com/aws/aws-sdk-go-v2/service/...
// - Forks: github.com/mycompany/aws-sdk-go-v2/service/...
// - Proxies: proxy.company.com/github.com/aws/aws-sdk-go-v2/service/...
// - Vendored: .../vendor/github.com/aws/aws-sdk-go-v2/service/...
// The key identifier "aws-sdk-go-v2/service/" is consistent across all these variants
// and uniquely identifies AWS SDK v2 service packages.
func isAWSSDKPackage(pkgPath string) bool {
	// Check for AWS SDK v2 service packages
	// We use Contains instead of HasPrefix to handle forks and proxies
	// The key identifier is "aws-sdk-go-v2/service/" which is consistent
	// across forks and proxies
	return strings.Contains(pkgPath, "aws-sdk-go-v2/service/")
}

// extractServiceNameFromPackage extracts the service name from a package path
// Example: "github.com/aws/aws-sdk-go-v2/service/s3" -> "s3"
func extractServiceNameFromPackage(pkgPath string) string {
	idx := strings.Index(pkgPath, "aws-sdk-go-v2/service/")
	if idx < 0 {
		return ""
	}
	servicePath := pkgPath[idx+len("aws-sdk-go-v2/service/"):]
	// Handle sub-packages (e.g., "s3/types" -> "s3")
	if slashIdx := strings.Index(servicePath, "/"); slashIdx >= 0 {
		return servicePath[:slashIdx]
	}
	return servicePath
}

// apiCallInfo contains information about an AWS SDK API call.
// Fields may be empty if the information cannot be extracted from the AST.
type apiCallInfo struct {
	// methodName is the API method being called (e.g., "ListBuckets", "ListTasks").
	// Empty if the call expression is not a selector expression.
	methodName string

	// serviceName is the AWS service name (e.g., "s3", "ecs", "dynamodb").
	// Empty if the result type doesn't come from an AWS SDK package.
	serviceName string

	// typeName is the full output type name (e.g., "ListBucketsOutput", "ListTasksOutput").
	// Empty if the result type is not a named type.
	typeName string
}

// extractAPICallInfo extracts API call information from a call expression
func extractAPICallInfo(callExpr *ast.CallExpr, resultType types.Type) apiCallInfo {
	info := apiCallInfo{}

	// Extract method name from call expression
	if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		info.methodName = sel.Sel.Name
	}

	// Extract type name and service name from result type
	if ptr, ok := resultType.(*types.Pointer); ok {
		resultType = ptr.Elem()
	}
	if named, ok := resultType.(*types.Named); ok {
		info.typeName = named.Obj().Name()
		pkg := named.Obj().Pkg()
		if pkg != nil {
			info.serviceName = extractServiceNameFromPackage(pkg.Path())
		}
	}

	return info
}

// buildErrorMessage constructs a concise, actionable error message.
// The message explains the problem, its impact, and the solution.
// For multi-field pagination (e.g., Route53), tokenFields contains multiple field names.
func buildErrorMessage(tokenFields []string, varName string, info apiCallInfo) string {
	var msg strings.Builder

	// Main problem description
	msg.WriteString("missing pagination handling for AWS SDK List API call")

	// Add context about the pagination token field(s)
	if len(tokenFields) > 0 {
		if len(tokenFields) == 1 {
			msg.WriteString(" (result has " + tokenFields[0] + " field)")
		} else {
			// Multi-field pagination (e.g., Route53)
			msg.WriteString(" (result has ")
			for i, field := range tokenFields {
				if i > 0 {
					if i == len(tokenFields)-1 {
						msg.WriteString(", and ")
					} else {
						msg.WriteString(", ")
					}
				}
				msg.WriteString(field)
			}
			msg.WriteString(" fields)")
		}
	}

	// Explain the impact
	msg.WriteString("\nWhen there are many results, only the first page is returned. Use ")

	// Suggest solution (context-aware if possible)
	if info.serviceName != "" && info.methodName != "" {
		msg.WriteString("New" + info.methodName + "Paginator")
	} else {
		msg.WriteString("a paginator")
	}

	msg.WriteString(" or loop with ")
	if varName != "" {
		msg.WriteString(varName + ".")
	}

	// Suggest field access pattern
	if len(tokenFields) > 0 {
		if len(tokenFields) == 1 {
			msg.WriteString(tokenFields[0] + ".")
		} else {
			// For multi-field, suggest checking any of the fields
			msg.WriteString(tokenFields[0] + " (or other pagination fields).")
		}
	}

	return msg.String()
}
