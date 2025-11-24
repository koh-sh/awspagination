package awspagination_test

import (
	"testing"

	"github.com/koh-sh/awspagination"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, awspagination.Analyzer, "test")
}

// TestIncludeTestFiles verifies that test files are analyzed when -include-tests=true
func TestIncludeTestFiles(t *testing.T) {
	// Enable test file analysis
	_ = awspagination.Analyzer.Flags.Set("include-tests", "true")
	defer func() {
		// Restore to default
		_ = awspagination.Analyzer.Flags.Set("include-tests", "false")
	}()

	testdata := analysistest.TestData()
	// Run analysis on testskip package - skip_test.go should be analyzed
	// and the want comments should be validated
	analysistest.Run(t, testdata, awspagination.Analyzer, "testskip")
}
