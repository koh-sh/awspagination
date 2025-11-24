package awspagination

import (
	"testing"
)

// TestGetPaginationTokenFields verifies that custom token fields are added to defaults
func TestGetPaginationTokenFields(t *testing.T) {
	// Save original config
	originalConfig := config
	defer func() { config = originalConfig }()

	// Test with no custom fields
	config = Config{}
	fields := getPaginationTokenFields()
	if len(fields) != len(defaultPaginationTokenFields) {
		t.Errorf("getPaginationTokenFields() without custom fields = %d fields, want %d",
			len(fields), len(defaultPaginationTokenFields))
	}

	// Verify default fields are present
	expectedDefaults := map[string]bool{
		"NextToken":             true,
		"NextMarker":            true,
		"Marker":                true,
		"NextContinuationToken": true,
		"ContinuationToken":     true,
		"NextPageToken":         true,
		"NextPageMarker":        true,
	}
	for _, field := range fields {
		if !expectedDefaults[field] {
			t.Errorf("Unexpected default field: %s", field)
		}
	}

	// Test with custom fields
	config = Config{
		CustomTokenFields: stringSliceFlag{"CustomToken", "MyPageToken"},
	}
	fields = getPaginationTokenFields()
	expectedTotal := len(defaultPaginationTokenFields) + 2
	if len(fields) != expectedTotal {
		t.Errorf("getPaginationTokenFields() with 2 custom fields = %d fields, want %d",
			len(fields), expectedTotal)
	}

	// Verify custom fields are included
	hasCustomToken := false
	hasMyPageToken := false
	for _, field := range fields {
		if field == "CustomToken" {
			hasCustomToken = true
		}
		if field == "MyPageToken" {
			hasMyPageToken = true
		}
	}
	if !hasCustomToken {
		t.Error("Custom field 'CustomToken' not found in pagination token fields")
	}
	if !hasMyPageToken {
		t.Error("Custom field 'MyPageToken' not found in pagination token fields")
	}
}

// TestExtractServiceNameFromPackage verifies service name extraction
func TestExtractServiceNameFromPackage(t *testing.T) {
	tests := []struct {
		name    string
		pkgPath string
		want    string
	}{
		{
			name:    "s3 service",
			pkgPath: "github.com/aws/aws-sdk-go-v2/service/s3",
			want:    "s3",
		},
		{
			name:    "ecs service",
			pkgPath: "github.com/aws/aws-sdk-go-v2/service/ecs",
			want:    "ecs",
		},
		{
			name:    "dynamodb with types subpackage",
			pkgPath: "github.com/aws/aws-sdk-go-v2/service/dynamodb/types",
			want:    "dynamodb",
		},
		{
			name:    "forked SDK",
			pkgPath: "github.com/mycompany/aws-sdk-go-v2/service/ec2",
			want:    "ec2",
		},
		{
			name:    "non-AWS package",
			pkgPath: "github.com/some/other/package",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractServiceNameFromPackage(tt.pkgPath)
			if got != tt.want {
				t.Errorf("extractServiceNameFromPackage(%q) = %q, want %q",
					tt.pkgPath, got, tt.want)
			}
		})
	}
}

// TestIsAWSSDKPackage verifies AWS SDK v2 package detection
func TestIsAWSSDKPackage(t *testing.T) {
	tests := []struct {
		name    string
		pkgPath string
		want    bool
	}{
		{
			name:    "s3 service",
			pkgPath: "github.com/aws/aws-sdk-go-v2/service/s3",
			want:    true,
		},
		{
			name:    "ecs service",
			pkgPath: "github.com/aws/aws-sdk-go-v2/service/ecs",
			want:    true,
		},
		{
			name:    "dynamodb service",
			pkgPath: "github.com/aws/aws-sdk-go-v2/service/dynamodb",
			want:    true,
		},
		{
			name:    "s3/types subpackage",
			pkgPath: "github.com/aws/aws-sdk-go-v2/service/s3/types",
			want:    true,
		},
		{
			name:    "forked SDK",
			pkgPath: "github.com/mycompany/aws-sdk-go-v2/service/ec2",
			want:    true,
		},
		{
			name:    "non-AWS package",
			pkgPath: "github.com/some/other/package",
			want:    false,
		},
		{
			name:    "AWS SDK v1",
			pkgPath: "github.com/aws/aws-sdk-go/service/s3",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAWSSDKPackage(tt.pkgPath)
			if got != tt.want {
				t.Errorf("isAWSSDKPackage(%q) = %v, want %v",
					tt.pkgPath, got, tt.want)
			}
		})
	}
}

// TestStringSliceFlag verifies the flag.Value implementation
func TestStringSliceFlag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantStr string
	}{
		{
			name:    "single value",
			input:   "s3",
			want:    []string{"s3"},
			wantStr: "s3",
		},
		{
			name:    "multiple values",
			input:   "s3,dynamodb,ecs",
			want:    []string{"s3", "dynamodb", "ecs"},
			wantStr: "s3,dynamodb,ecs",
		},
		{
			name:    "empty string",
			input:   "",
			want:    []string{},
			wantStr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var flag stringSliceFlag
			err := flag.Set(tt.input)
			if err != nil {
				t.Errorf("stringSliceFlag.Set(%q) error = %v", tt.input, err)
				return
			}

			if len(flag) != len(tt.want) {
				t.Errorf("stringSliceFlag.Set(%q) length = %d, want %d",
					tt.input, len(flag), len(tt.want))
				return
			}

			for i, v := range flag {
				if v != tt.want[i] {
					t.Errorf("stringSliceFlag.Set(%q)[%d] = %q, want %q",
						tt.input, i, v, tt.want[i])
				}
			}

			gotStr := flag.String()
			if gotStr != tt.wantStr {
				t.Errorf("stringSliceFlag.String() = %q, want %q", gotStr, tt.wantStr)
			}
		})
	}
}

// TestNew verifies the New function for module plugin integration
func TestNew(t *testing.T) {
	// Save original config
	originalConfig := config
	defer func() { config = originalConfig }()

	tests := []struct {
		name     string
		settings any
		want     Settings
		wantErr  bool
	}{
		{
			name: "valid settings with custom fields",
			settings: map[string]any{
				"custom-fields": []any{"MyToken", "CustomNextToken"},
				"include-tests": true,
			},
			want: Settings{
				CustomFields: []string{"MyToken", "CustomNextToken"},
				IncludeTests: true,
			},
			wantErr: false,
		},
		{
			name: "valid settings without custom fields",
			settings: map[string]any{
				"include-tests": false,
			},
			want: Settings{
				CustomFields: nil,
				IncludeTests: false,
			},
			wantErr: false,
		},
		{
			name:     "empty settings",
			settings: map[string]any{},
			want: Settings{
				CustomFields: nil,
				IncludeTests: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config before each test
			config = Config{}

			analyzers, err := New(tt.settings)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify that analyzers were returned
			if len(analyzers) != 1 {
				t.Errorf("New() returned %d analyzers, want 1", len(analyzers))
				return
			}

			if analyzers[0] != Analyzer {
				t.Error("New() did not return the expected Analyzer")
			}

			// Verify that config was updated correctly
			if len(config.CustomTokenFields) != len(tt.want.CustomFields) {
				t.Errorf("config.CustomTokenFields length = %d, want %d",
					len(config.CustomTokenFields), len(tt.want.CustomFields))
			}

			for i, field := range tt.want.CustomFields {
				if i < len(config.CustomTokenFields) && config.CustomTokenFields[i] != field {
					t.Errorf("config.CustomTokenFields[%d] = %q, want %q",
						i, config.CustomTokenFields[i], field)
				}
			}

			if config.IncludeTests != tt.want.IncludeTests {
				t.Errorf("config.IncludeTests = %v, want %v",
					config.IncludeTests, tt.want.IncludeTests)
			}
		})
	}
}
