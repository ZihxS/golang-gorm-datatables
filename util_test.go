package datatables

import (
	"reflect"
	"regexp"
	"testing"
)

func TestExtractFields(t *testing.T) {
	tests := []struct {
		name     string
		clause   string
		expected []string
	}{
		{"empty_string", "", []string{}},
		{"no_parentheses", "field_1, field_2", []string{"field_1", "field_2"}},
		{"with_parentheses", "(field_1, field_2)", []string{"field_1", "field_2"}},
		{"multiple_fields", "field_1, field_2, field_3", []string{"field_1", "field_2", "field_3"}},
		{"leading_or_trailing_whitespace", " (field_1, field_2) ", []string{"field_1", "field_2"}},
		{"whitespace_between_fields", "field_1 , field_2 , field_3", []string{"field_1", "field_2", "field_3"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := extractFields(test.clause)
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("expected %#v, got %#v", test.expected, actual)
			}
		})
	}
}

func TestQM(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic_string",
			input:    "SELECT * FROM users",
			expected: regexp.QuoteMeta("SELECT * FROM users"),
		},
		{
			name:     "string_with_special_characters",
			input:    "SELECT * FROM users WHERE id = ?",
			expected: regexp.QuoteMeta("SELECT * FROM users WHERE id = ?"),
		},
		{
			name:     "empty_string",
			input:    "",
			expected: regexp.QuoteMeta(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qm(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNormalizeResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    []map[string]any
		expected []map[string]any
	}{
		{
			name: "convert_int64_to_int",
			input: []map[string]any{
				{"id": int64(1), "name": "John"},
				{"id": int64(2), "name": "Jane"},
			},
			expected: []map[string]any{
				{"id": 1, "name": "John"},
				{"id": 2, "name": "Jane"},
			},
		},
		{
			name: "no_conversion_needed",
			input: []map[string]any{
				{"id": 1, "name": "John"},
				{"id": 2, "name": "Jane"},
			},
			expected: []map[string]any{
				{"id": 1, "name": "John"},
				{"id": 2, "name": "Jane"},
			},
		},
		{
			name: "mixed_types",
			input: []map[string]any{
				{"id": int64(1), "age": 25, "active": true},
				{"id": int64(2), "age": 30, "active": false},
			},
			expected: []map[string]any{
				{"id": 1, "age": 25, "active": true},
				{"id": 2, "age": 30, "active": false},
			},
		},
		{
			name:     "empty_input",
			input:    []map[string]any{},
			expected: []map[string]any{},
		},
		{
			name:     "nil_input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeResponse(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
