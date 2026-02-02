package appengine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// FunctionSchema represents the discovered schema of a function
type FunctionSchema struct {
	FunctionName string
	AppID        string
	Fields       []SchemaField
	ErrorMessage string
}

// SchemaField represents a field in the function schema
type SchemaField struct {
	Name     string
	Type     string
	Required bool
	Hint     string
}

// DiscoverSchema attempts to discover the schema of a function by calling it with an empty payload
func (h *FunctionHandler) DiscoverSchema(appID, functionName string) (*FunctionSchema, error) {
	// Try with empty payload to trigger validation
	req := &FunctionInvokeRequest{
		Method:       "POST",
		AppID:        appID,
		FunctionName: functionName,
		Payload:      "{}",
		Headers:      make(map[string]string),
	}

	resp, err := h.InvokeFunction(req)
	if err != nil {
		// If we got a non-validation error, return it
		if !strings.Contains(err.Error(), "validation") &&
			!strings.Contains(err.Error(), "Invalid input") &&
			!strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "required") &&
			!strings.Contains(err.Error(), "Required") {
			return nil, err
		}
	}

	schema := &FunctionSchema{
		FunctionName: functionName,
		AppID:        appID,
		Fields:       []SchemaField{},
	}

	// Parse the response body to extract schema information
	if resp != nil && resp.Body != "" {
		var bodyData map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Body), &bodyData); err == nil {
			if errorMsg, ok := bodyData["error"].(string); ok {
				schema.ErrorMessage = errorMsg
				schema.Fields = parseSchemaFromError(errorMsg)

				// If no fields discovered and error suggests empty input, try with minimal object
				if len(schema.Fields) == 0 && strings.Contains(errorMsg, "must not be empty") {
					req.Payload = `{"test":"value"}`
					resp2, _ := h.InvokeFunction(req)
					if resp2 != nil && resp2.Body != "" {
						var bodyData2 map[string]interface{}
						if err := json.Unmarshal([]byte(resp2.Body), &bodyData2); err == nil {
							if errorMsg2, ok := bodyData2["error"].(string); ok {
								schema.ErrorMessage = errorMsg2
								schema.Fields = parseSchemaFromError(errorMsg2)
							}
						}
					}
				}
			}
		}
	}

	return schema, nil
}

// parseSchemaFromError extracts schema information from error messages
func parseSchemaFromError(errorMsg string) []SchemaField {
	var fields []SchemaField

	// Pattern 0: "Action input must not be empty" - try with minimal object
	if strings.Contains(errorMsg, "Action input must not be empty") {
		// Return empty - this function needs a non-empty object to reveal fields
		return fields
	}

	// Pattern 0.5: Try to parse as JSON structure first (for Zod-style errors)
	// "Invalid input: { \"field\": [\"error\"] }"
	if strings.Contains(errorMsg, "Invalid input:") && strings.Contains(errorMsg, "{") {
		// Extract the JSON part
		jsonStart := strings.Index(errorMsg, "{")
		if jsonStart != -1 {
			jsonStr := errorMsg[jsonStart:]
			var errorFields map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &errorFields); err == nil {
				for fieldName, fieldError := range errorFields {
					field := SchemaField{
						Name:     fieldName,
						Type:     "unknown",
						Required: true,
					}

					// Try to extract type from error message array
					if errArray, ok := fieldError.([]interface{}); ok && len(errArray) > 0 {
						if errMsg, ok := errArray[0].(string); ok {
							field.Hint = errMsg
							// Extract type from error message
							if strings.Contains(errMsg, "expected object") {
								field.Type = "object"
							} else if strings.Contains(errMsg, "expected string") {
								field.Type = "string"
							} else if strings.Contains(errMsg, "expected array") {
								field.Type = "array"
							} else if strings.Contains(errMsg, "expected boolean") {
								field.Type = "boolean"
							} else if strings.Contains(errMsg, "expected number") {
								field.Type = "number"
							}
						}
					}

					fields = append(fields, field)
				}
				return fields
			}
		}
	}

	// Pattern 1: "Input fields 'query' are missing."
	re1 := regexp.MustCompile(`Input fields? ['"]([^'"]+)['"] (?:are|is) missing`)
	if matches := re1.FindStringSubmatch(errorMsg); len(matches) > 1 {
		fieldNames := strings.Split(matches[1], ",")
		for _, name := range fieldNames {
			fields = append(fields, SchemaField{
				Name:     strings.TrimSpace(name),
				Type:     "unknown",
				Required: true,
			})
		}
		return fields
	}

	// Pattern 2: "connectionId - project - issueType - components - summary - description"
	re2 := regexp.MustCompile(`wrong fields?: ['"]([^'"]+)['"]`)
	if matches := re2.FindStringSubmatch(errorMsg); len(matches) > 1 {
		fieldNames := strings.Split(matches[1], " - ")
		for _, name := range fieldNames {
			fields = append(fields, SchemaField{
				Name:     strings.TrimSpace(name),
				Type:     "unknown",
				Required: true,
			})
		}
		return fields
	}

	// Pattern 3: "connection: Required\n   - channel: Required\n   - message: Must be defined"
	re3 := regexp.MustCompile(`(?m)^\s*-?\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:\s*(Required|Must be defined|Invalid input)`)
	matches := re3.FindAllStringSubmatch(errorMsg, -1)
	if len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				fields = append(fields, SchemaField{
					Name:     match[1],
					Type:     "unknown",
					Required: true,
				})
			}
		}
		return fields
	}

	// Pattern 4: "observable": ["Invalid input: expected object, received undefined"]
	re4 := regexp.MustCompile(`"([a-zA-Z_][a-zA-Z0-9_]*)"\s*:\s*\[?\s*"[^"]*expected\s+(\w+),\s*received`)
	matches = re4.FindAllStringSubmatch(errorMsg, -1)
	if len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 2 {
				fields = append(fields, SchemaField{
					Name:     match[1],
					Type:     match[2],
					Required: true,
				})
			}
		}
		return fields
	}

	// Pattern 5: Line-by-line validation errors
	lines := strings.Split(errorMsg, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			line = strings.TrimPrefix(line, "- ")
		}

		// Extract field name and type hint
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			fieldName := strings.TrimSpace(parts[0])
			hint := strings.TrimSpace(parts[1])

			// Skip if this doesn't look like a field name
			if fieldName == "" || !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(fieldName) {
				continue
			}

			field := SchemaField{
				Name:     fieldName,
				Type:     "unknown",
				Required: true,
				Hint:     hint,
			}

			// Try to extract type information from hint
			if strings.Contains(hint, "string") {
				field.Type = "string"
			} else if strings.Contains(hint, "object") {
				field.Type = "object"
			} else if strings.Contains(hint, "array") {
				field.Type = "array"
			} else if strings.Contains(hint, "boolean") {
				field.Type = "boolean"
			} else if strings.Contains(hint, "number") {
				field.Type = "number"
			}

			fields = append(fields, field)
		}
	}

	return fields
}

// FormatSchema formats a schema for display
func (s *FunctionSchema) FormatSchema() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Function: %s/%s\n\n", s.AppID, s.FunctionName))

	if len(s.Fields) == 0 {
		sb.WriteString("Unable to discover schema fields.\n")
		if s.ErrorMessage != "" {
			sb.WriteString("\nRaw error message:\n")
			sb.WriteString(s.ErrorMessage)
			sb.WriteString("\n")
		}
		return sb.String()
	}

	sb.WriteString("Required Fields:\n")
	maxNameLen := 0
	maxTypeLen := 0
	for _, field := range s.Fields {
		if len(field.Name) > maxNameLen {
			maxNameLen = len(field.Name)
		}
		if len(field.Type) > maxTypeLen {
			maxTypeLen = len(field.Type)
		}
	}

	for _, field := range s.Fields {
		sb.WriteString(fmt.Sprintf("  %-*s  %-*s", maxNameLen, field.Name, maxTypeLen, field.Type))
		if field.Hint != "" {
			sb.WriteString(fmt.Sprintf("  %s", field.Hint))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\nExample payload:\n")
	sb.WriteString(s.GenerateExamplePayload())
	sb.WriteString("\n")

	return sb.String()
}

// GenerateExamplePayload generates an example JSON payload based on discovered schema
func (s *FunctionSchema) GenerateExamplePayload() string {
	if len(s.Fields) == 0 {
		return "{}"
	}

	payload := make(map[string]interface{})
	for _, field := range s.Fields {
		switch field.Type {
		case "string":
			payload[field.Name] = "..."
		case "object":
			payload[field.Name] = map[string]interface{}{}
		case "array":
			payload[field.Name] = []interface{}{}
		case "boolean":
			payload[field.Name] = false
		case "number":
			payload[field.Name] = 0
		default:
			payload[field.Name] = "..."
		}
	}

	jsonBytes, err := json.MarshalIndent(payload, "  ", "  ")
	if err != nil {
		return "{}"
	}

	return "  " + string(jsonBytes)
}
