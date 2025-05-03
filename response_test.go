package datatables

import (
	"reflect"
	"testing"
)

func TestApplyCustomColumns(t *testing.T) {
	tests := []struct {
		name         string
		customCols   []func(map[string]any) map[string]any
		data         []map[string]any
		expectedData []map[string]any
	}{
		{
			name:         "no_custom_columns",
			customCols:   nil,
			data:         []map[string]any{{"key": "value"}},
			expectedData: []map[string]any{{"key": "value"}},
		},
		{
			name: "one_custom_column",
			customCols: []func(map[string]any) map[string]any{
				func(data map[string]any) map[string]any {
					data["newKey"] = "newValue"
					return data
				},
			},
			data:         []map[string]any{{"key": "value"}},
			expectedData: []map[string]any{{"key": "value", "newKey": "newValue"}},
		},
		{
			name: "multiple_custom_columns",
			customCols: []func(map[string]any) map[string]any{
				func(data map[string]any) map[string]any { data["newKey_1"] = "newValue_1"; return data },
				func(data map[string]any) map[string]any { data["newKey_2"] = "newValue_2"; return data },
			},
			data:         []map[string]any{{"key": "value"}},
			expectedData: []map[string]any{{"key": "value", "newKey_1": "newValue_1", "newKey_2": "newValue_2"}},
		},
		{
			name:         "nil_data",
			customCols:   nil,
			data:         nil,
			expectedData: nil,
		},
		{
			name:         "empty_data",
			customCols:   nil,
			data:         []map[string]any{},
			expectedData: []map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := New(nil)
			dt.customCols = tt.customCols
			dt.applyCustomColumns(tt.data)
			if !reflect.DeepEqual(tt.data, tt.expectedData) {
				t.Errorf("expected data to be %#v, but got %#v", tt.expectedData, tt.data)
			}
		})
	}
}

func TestApplyRowAttributes(t *testing.T) {
	tests := []struct {
		name         string
		rowIdFunc    func(map[string]any) string
		rowClass     string
		rowDataFunc  func(map[string]any) map[string]any
		expectedData []map[string]any
	}{
		{
			name:         "no_row_attributes_set",
			expectedData: []map[string]any{{"key_1": "value_1"}, {"key_2": "value_2"}},
		},
		{
			name:      "row_id_function_set",
			rowIdFunc: func(row map[string]any) string { return "row-id" },
			expectedData: []map[string]any{
				{"key_1": "value_1", datatableRowID: "row-id"},
				{"key_2": "value_2", datatableRowID: "row-id"},
			},
		},
		{
			name:     "row_class_set",
			rowClass: "row-class",
			expectedData: []map[string]any{
				{"key_1": "value_1", datatableRowClass: "row-class"},
				{"key_2": "value_2", datatableRowClass: "row-class"},
			},
		},
		{
			name: "row_data_function_set",
			rowDataFunc: func(row map[string]any) map[string]any {
				return map[string]any{"custom-key": "custom-value"}
			},
			expectedData: []map[string]any{
				{"key_1": "value_1", datatableRowDataPrefix + "custom-key": "custom-value"},
				{"key_2": "value_2", datatableRowDataPrefix + "custom-key": "custom-value"},
			},
		},
		{
			name:      "all_row_attributes_set",
			rowIdFunc: func(row map[string]any) string { return "row-id" },
			rowClass:  "row-class",
			rowDataFunc: func(row map[string]any) map[string]any {
				return map[string]any{"custom-key": "custom-value"}
			},
			expectedData: []map[string]any{
				{"key_1": "value_1", datatableRowID: "row-id", datatableRowClass: "row-class", datatableRowDataPrefix + "custom-key": "custom-value"},
				{"key_2": "value_2", datatableRowID: "row-id", datatableRowClass: "row-class", datatableRowDataPrefix + "custom-key": "custom-value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := New(nil)
			dt.rowIdFunc = tt.rowIdFunc
			dt.rowClass = tt.rowClass
			dt.rowDataFunc = tt.rowDataFunc

			data := []map[string]any{{"key_1": "value_1"}, {"key_2": "value_2"}}
			dt.applyRowAttributes(data)
			if !reflect.DeepEqual(data, tt.expectedData) {
				t.Errorf("expected data to be %+v, but got %+v", tt.expectedData, data)
			}
		})
	}
}

func TestGetFilteredColumns(t *testing.T) {
	tests := []struct {
		name            string
		selectedColumns []string
		columns         []Column
		expected        []Column
	}{
		{
			name:            "with_selected_columns",
			selectedColumns: []string{"col_1", "col_2"},
			columns:         []Column{{Name: "col_1", Data: "col_1"}, {Name: "col_2", Data: "col_2"}, {Name: "col_3", Data: "col_3"}},
			expected:        []Column{{Name: "col_1", Data: "col_1"}, {Name: "col_2", Data: "col_2"}},
		},
		{
			name:            "with_no_selected_columns",
			selectedColumns: []string{},
			columns:         []Column{{Name: "col_1", Data: "col_1"}, {Name: "col_2", Data: "col_2"}, {Name: "col_3", Data: "col_3"}},
			expected:        []Column{{Name: "col_1", Data: "col_1"}, {Name: "col_2", Data: "col_2"}, {Name: "col_3", Data: "col_3"}},
		},
		{
			name:            "with_empty_columns",
			selectedColumns: []string{},
			columns:         []Column{},
			expected:        []Column{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := New(nil)
			dt.selectedColumns = tt.selectedColumns
			dt.columns = tt.columns
			dt.initColumnsMap()

			result := dt.getFilteredColumns()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TestGetFilteredColumns %#v: expected %#v, got %#v", tt.name, tt.expected, result)
			}
		})
	}
}
