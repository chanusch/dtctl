package appengine

import (
	"strings"
	"testing"
)

func TestParseSchemaFromError(t *testing.T) {
	tests := []struct {
		name       string
		errorMsg   string
		wantFields int
	}{
		{
			name:       "empty action input message",
			errorMsg:   "Action input must not be empty",
			wantFields: 0,
		},
		{
			name:       "zod-style JSON validation error",
			errorMsg:   `Invalid input: { "query": ["Required"], "timeframe": ["Required"] }`,
			wantFields: 2,
		},
		{
			name:       "no schema information",
			errorMsg:   "Internal server error occurred",
			wantFields: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := parseSchemaFromError(tt.errorMsg)

			if len(fields) != tt.wantFields {
				t.Errorf("parseSchemaFromError() got %d fields, want %d", len(fields), tt.wantFields)
			}
		})
	}
}

func TestFunctionSchema_FormatSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema *FunctionSchema
		checks []string
	}{
		{
			name: "schema with fields",
			schema: &FunctionSchema{
				FunctionName: "execute-dql-query",
				AppID:        "dynatrace.automations",
				Fields: []SchemaField{
					{Name: "query", Type: "string", Required: true, Hint: "DQL query to execute"},
					{Name: "timeout", Type: "number", Required: true, Hint: "Query timeout in seconds"},
				},
			},
			checks: []string{
				"execute-dql-query",
				"dynatrace.automations",
				"query",
				"string",
				"timeout",
				"number",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.schema.FormatSchema()

			for _, check := range tt.checks {
				if !strings.Contains(strings.ToLower(output), strings.ToLower(check)) {
					t.Errorf("FormatSchema() output should contain %q, got:\n%s", check, output)
				}
			}
		})
	}
}

func TestFunctionSchema_GenerateExamplePayload(t *testing.T) {
	tests := []struct {
		name     string
		schema   *FunctionSchema
		wantJSON bool
	}{
		{
			name: "schema with various field types",
			schema: &FunctionSchema{
				FunctionName: "test-function",
				AppID:        "test.app",
				Fields: []SchemaField{
					{Name: "query", Type: "string", Required: true},
					{Name: "count", Type: "number", Required: false},
				},
			},
			wantJSON: true,
		},
		{
			name: "empty schema",
			schema: &FunctionSchema{
				FunctionName: "empty-function",
				AppID:        "test.app",
				Fields:       []SchemaField{},
			},
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := tt.schema.GenerateExamplePayload()

			if tt.wantJSON {
				if !strings.HasPrefix(strings.TrimSpace(payload), "{") {
					t.Errorf("GenerateExamplePayload() should start with '{'")
				}
			}
		})
	}
}
