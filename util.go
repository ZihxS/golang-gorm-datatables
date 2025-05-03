package datatables

import (
	"regexp"
	"strings"
)

// extractFields takes a SQL GROUP BY or HAVING clause as input and returns
// a slice of strings representing the fields in the clause. The input string
// should not include the keyword "GROUP BY" or "HAVING". If the clause is
// empty, an empty slice is returned.
//
// The function removes any leading or trailing parentheses and whitespace
// from the input string, splits the string on commas, trims any remaining
// whitespace from the resulting fields, and returns the fields as a slice.
func extractFields(clause string) []string {
	if clause == "" {
		return []string{}
	}

	openParenIndex := strings.Index(clause, "(")
	if openParenIndex != -1 {
		clause = clause[openParenIndex+1:]
	}

	closeParenIndex := strings.Index(clause, ")")
	if closeParenIndex != -1 {
		clause = clause[:closeParenIndex]
	}

	fields := strings.Split(clause, ",")
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}

	return fields
}

// qm takes a string as input and returns a string with any special
// characters properly escaped for use in a regular expression. This
// function is useful for protecting against user input that may contain
// special characters. It uses the regexp.QuoteMeta function from the
// Go standard library.
func qm(str string) string {
	return regexp.QuoteMeta(str)
}

// normalizeResponse takes a slice of maps as input and returns a new slice of
// maps where all int64 values are converted to int. This is useful for
// preparing data for JSON encoding, since the encoding/json package does not
// support int64. The function returns nil if the input data is nil.
func normalizeResponse(data []map[string]any) []map[string]any {
	if data == nil {
		return nil
	}
	normalized := make([]map[string]any, len(data))
	for i, row := range data {
		normalized[i] = make(map[string]any)
		for key, value := range row {
			switch v := value.(type) {
			case int64:
				normalized[i][key] = int(v)
			default:
				normalized[i][key] = value
			}
		}
	}
	return normalized
}

// normalizeResponseMake takes a map as input and returns a new map where all int values are
// converted to int64. If the value is a slice of maps, it calls normalizeDataArray on the slice.
// This function is used to normalize the "data" field in the DataTables response.
func normalizeResponseMake(data map[string]any) map[string]any {
	normalized := make(map[string]any)
	for k, v := range data {
		switch v := v.(type) {
		case int:
			normalized[k] = int64(v)
		case []map[string]any:
			normalized[k] = normalizeDataArray(v)
		default:
			normalized[k] = v
		}
	}
	return normalized
}

// normalizeDataArray takes a slice of maps as input and returns a new slice of
// maps where all int values are converted to int64. It is used to normalize the
// "data" field in the DataTables response.
func normalizeDataArray(data []map[string]any) []map[string]any {
	result := make([]map[string]any, len(data))
	for i, row := range data {
		normalizedRow := make(map[string]any)
		for k, v := range row {
			switch v := v.(type) {
			case int:
				normalizedRow[k] = int64(v)
			default:
				normalizedRow[k] = v
			}
		}
		result[i] = normalizedRow
	}
	return result
}
