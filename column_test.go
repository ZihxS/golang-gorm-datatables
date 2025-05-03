package datatables

import (
	"reflect"
	"testing"
)

func TestInitColumnsMap(t *testing.T) {
	dt := New(nil)
	dt.columns = []Column{
		{Name: "ID", Data: "id", Searchable: true, Orderable: true},
		{Name: "Name", Data: "name", Searchable: false, Orderable: false},
	}

	dt.initColumnsMap()

	if len(dt.columnsMap) != 2 {
		t.Errorf("expected columnsMap to have 2 entries, got %d", len(dt.columnsMap))
	}
	for _, col := range dt.columns {
		if dt.columnsMap[col.Data].Name != col.Name {
			t.Errorf("columnsMap was not initialized correctly for column '%s'", col.Data)
		}
	}
}

func TestIsColumnAllowed(t *testing.T) {
	tests := []struct {
		name           string
		whitelist      map[string]bool
		blacklist      map[string]bool
		column         string
		expectedResult bool
	}{
		{"no_whitelist_or_blacklist", nil, nil, "id", true},
		{"whitelist_allows_column", map[string]bool{"id": true}, nil, "id", true},
		{"whitelist_disallows_column", map[string]bool{"id": true}, nil, "name", false},
		{"blacklist_disallows_column", nil, map[string]bool{"name": true}, "name", false},
		{"blacklist_allows_column", nil, map[string]bool{"name": true}, "id", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := New(nil)
			dt.whitelistColumns = tt.whitelist
			dt.blacklistColumns = tt.blacklist

			result := dt.isColumnAllowed(tt.column)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestAddColumn(t *testing.T) {
	dt := New(nil)

	dt.AddColumn(Column{
		Name:       "ID",
		Data:       "id",
		Searchable: true,
		Orderable:  true,
		RenderFunc: nil,
	})
	if len(dt.columns) != 1 || dt.columns[0].Name != "ID" {
		t.Errorf("column was not added correctly")
	}
	if _, exists := dt.columnsMap["id"]; !exists {
		t.Errorf("column was not added to columnsMap")
	}

	dt.AddColumn(Column{
		Name:       "ID",
		Data:       "id",
		Searchable: true,
		Orderable:  true,
		RenderFunc: nil,
	})
	if len(dt.columns) != 1 {
		t.Errorf("duplicate column should not be added")
	}
}

func TestAddColumns(t *testing.T) {
	dt := New(nil)

	columns := []Column{
		{Name: "ID", Data: "id", Searchable: true, Orderable: true},
		{Name: "Name", Data: "name", Searchable: false, Orderable: false},
	}

	dt.AddColumns(columns...)
	if len(dt.columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(dt.columns))
	}
	for i, col := range columns {
		if dt.columns[i].Name != col.Name || dt.columns[i].Data != col.Data {
			t.Errorf("columns were not added correctly")
		}
	}
}

func TestEditColumn(t *testing.T) {
	dt := New(nil)
	dt.AddColumn(Column{
		Name:       "ID",
		Data:       "id",
		Searchable: true,
		Orderable:  true,
		RenderFunc: nil,
	})

	editFunc := func(value any) any { return value.(string) + "-edited" }
	dt.EditColumn("id", editFunc)

	if dt.columnsMap["id"].RenderFunc == nil {
		t.Errorf("render function was not updated")
	}

	row := map[string]any{"id": "123"}
	result := dt.columnsMap["id"].RenderFunc(row)
	if result != "123-edited" {
		t.Errorf("render function did not work as expected, got %v", result)
	}
}

func TestRemoveColumn(t *testing.T) {
	dt := New(nil)
	dt.AddColumn(Column{
		Name:       "ID",
		Data:       "id",
		Searchable: true,
		Orderable:  true,
		RenderFunc: nil,
	})
	dt.AddColumn(Column{
		Name:       "Name",
		Data:       "name",
		Searchable: true,
		Orderable:  true,
		RenderFunc: nil,
	})
	dt.selectedColumns = []string{"id", "name"}

	dt.RemoveColumn("id")
	if !reflect.DeepEqual(dt.selectedColumns, []string{"name"}) {
		t.Errorf("column was not removed correctly, got %v", dt.selectedColumns)
	}

	dt.RemoveColumn("name")
	if len(dt.selectedColumns) > 0 {
		t.Errorf("last column was not removed correctly, got %v", dt.selectedColumns)
	}

	dt.RemoveColumn("non_existent_column")
	if len(dt.selectedColumns) > 0 {
		t.Errorf("removing a non-existent column should not modify selectedColumns, got %v", dt.selectedColumns)
	}

	dt.RemoveColumn("id")
	if len(dt.selectedColumns) > 0 {
		t.Errorf("removing from an empty selectedColumns slice should not modify it, got %v", dt.selectedColumns)
	}

	dt.columns = []Column{
		{Name: "ID", Data: "id", Searchable: true, Orderable: true},
		{Name: "Name", Data: "name", Searchable: false, Orderable: false},
	}
	dt.RemoveColumn("id")
	if !reflect.DeepEqual(dt.columns, []Column{{Name: "Name", Data: "name", Searchable: false, Orderable: false}}) {
		t.Errorf("column was not removed correctly, got %v", dt.columns)
	}
}

func TestWhitelistAndBlacklistColumns(t *testing.T) {
	tests := []struct {
		name     string
		method   func(dt *DataTable, columns ...string)
		columns  []string
		expected map[string]bool
	}{
		{
			name: "whitelist_column",
			method: func(dt *DataTable, columns ...string) {
				dt.WhitelistColumn(columns...)
			},
			columns:  []string{"id", "name"},
			expected: map[string]bool{"id": true, "name": true},
		},
		{
			name: "blacklist_column",
			method: func(dt *DataTable, columns ...string) {
				dt.BlacklistColumn(columns...)
			},
			columns:  []string{"id", "name"},
			expected: map[string]bool{"id": true, "name": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := New(nil)
			tt.method(dt, tt.columns...)

			if tt.name == "whitelist_column" && !reflect.DeepEqual(dt.whitelistColumns, tt.expected) {
				t.Errorf("whitelist columns were not set correctly, got %v", dt.whitelistColumns)
			}
			if tt.name == "blacklist_column" && !reflect.DeepEqual(dt.blacklistColumns, tt.expected) {
				t.Errorf("blacklist columns were not set correctly, got %v", dt.blacklistColumns)
			}
		})
	}
}

func TestFinalizeResponseColumns(t *testing.T) {
	tests := []struct {
		name         string
		data         []map[string]any
		selectedCols []string
		expected     []map[string]any
	}{
		{
			name:     "empty_data",
			data:     []map[string]any{},
			expected: []map[string]any{},
		},
		{
			name:         "single_row_and_single_column",
			data:         []map[string]any{{"col1": "val1"}},
			selectedCols: []string{"col1"},
			expected:     []map[string]any{{"col1": "val1"}},
		},
		{
			name: "multiple_rows_and_multiple_columns",
			data: []map[string]any{
				{"col1": "val1", "col2": "val2"},
				{"col1": "val3", "col2": "val4"},
			},
			selectedCols: []string{"col1"},
			expected: []map[string]any{
				{"col1": "val1"},
				{"col1": "val3"},
			},
		},
		{
			name: "selected_columns",
			data: []map[string]any{
				{"col1": "val1", "col2": "val2"},
				{"col1": "val3", "col2": "val4"},
			},
			selectedCols: []string{"col1", "col2"},
			expected: []map[string]any{
				{"col1": "val1", "col2": "val2"},
				{"col1": "val3", "col2": "val4"},
			},
		},
		{
			name: "unselected_columns",
			data: []map[string]any{
				{"col1": "val1", "col2": "val2"},
				{"col1": "val3", "col2": "val4"},
			},
			selectedCols: []string{},
			expected: []map[string]any{
				{},
				{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := &DataTable{selectedColumns: tt.selectedCols}
			actual := dt.FinalizeResponseColumns(tt.data)
			if len(actual) != len(tt.expected) {
				t.Errorf("expected %d rows, got %d", len(tt.expected), len(actual))
			}
			for i, row := range actual {
				if len(row) != len(tt.expected[i]) {
					t.Errorf("expected %d columns, got %d", len(tt.expected[i]), len(row))
				}
				for key, value := range row {
					if tt.expected[i][key] != value {
						t.Errorf("expected %v, got %v", tt.expected[i][key], value)
					}
				}
			}
		})
	}
}
