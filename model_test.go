package datatables

import (
	"reflect"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestModel(t *testing.T) {
	dt := New(nil)
	model := struct{ Name string }{}

	result := dt.Model(model)
	if result.model != model {
		t.Errorf("expected model to be set, got %v", result.model)
	}
}

func TestReq(t *testing.T) {
	dt := New(nil)
	req := Request{
		Columns: []ColumnRequest{
			{Name: "ID", Data: "id", Searchable: true, Orderable: true},
			{Name: "Name", Data: "name", Searchable: false, Orderable: false},
		},
	}

	result := dt.Req(req)
	if len(result.columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(result.columns))
	}
	for i, col := range req.Columns {
		if result.columns[i].Name != col.Name || result.columns[i].Data != col.Data {
			t.Errorf("columns were not added correctly")
		}
	}
}

func TestOnly(t *testing.T) {
	dt := New(nil)
	columns := []string{"id", "name"}

	result := dt.Only(columns...)
	if !reflect.DeepEqual(result.selectedColumns, columns) {
		t.Errorf("expected selectedColumns to be %v, got %v", columns, result.selectedColumns)
	}
}

func TestWith(t *testing.T) {
	dt := New(nil)
	relations := []string{"Profile", "Address"}

	result := dt.With(relations...)
	if !reflect.DeepEqual(result.relations, relations) {
		t.Errorf("expected relations to be %v, got %v", relations, result.relations)
	}
}

func TestWithData(t *testing.T) {
	dt := New(nil)
	key, value := "key", "value"

	result := dt.WithData(key, value)
	if result.additionalData[key] != value {
		t.Errorf("expected additionalData[%s] to be %v, got %v", key, value, result.additionalData[key])
	}
}

func TestWithNumber(t *testing.T) {
	dt := New(nil)

	result := dt.WithNumber()
	if len(result.columns) != 1 || result.columns[0].Name != "No" {
		t.Errorf("expected 'No' column to be added, got %v", result.columns)
	}
}

func TestFilter(t *testing.T) {
	dt := New(nil)
	filterFunc := func(db *gorm.DB) *gorm.DB { return db.Where("active = ?", true) }

	result := dt.Filter(filterFunc)
	if len(result.filters) != 1 {
		t.Errorf("expected 1 filter, got %d", len(result.filters))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*DataTable)
		wantErr bool
	}{
		{
			name:    "invalid_case_no_model_or_tx",
			setup:   func(dt *DataTable) {},
			wantErr: true,
		},
		{
			name: "invalid_case_nil_statement",
			setup: func(dt *DataTable) {
				dt.tx = &gorm.DB{}
			},
			wantErr: true,
		},
		{
			name: "invalid_case_zero_draw_and_zero_columns",
			setup: func(dt *DataTable) {
				dt.tx = &gorm.DB{Statement: &gorm.Statement{Model: struct{}{}}}
				dt.Req(Request{})
			},
			wantErr: true,
		},
		{
			name: "invalid_case_regex_pattern",
			setup: func(dt *DataTable) {
				dt.tx = &gorm.DB{Statement: &gorm.Statement{Model: struct{}{}}}
				dt.Req(Request{
					Draw: 1,
					Columns: []ColumnRequest{
						{Name: "ID", Data: "id", Searchable: true, Orderable: true},
					},
					Search: Search{
						Value: "{2,5}",
						Regex: true,
					},
				})

			},
			wantErr: true,
		},
		{
			name: "valid_case_with_a_proper_request",
			setup: func(dt *DataTable) {
				dt.tx = &gorm.DB{Statement: &gorm.Statement{Model: struct{}{}}}
				dt.Req(Request{
					Draw: 1,
					Columns: []ColumnRequest{
						{Name: "ID", Data: "id", Searchable: true, Orderable: true},
					},
				})
			},
			wantErr: false,
		},
		{
			name: "valid_case_with_a_proper_request_using_table_expr",
			setup: func(dt *DataTable) {
				dt.tx = &gorm.DB{Statement: &gorm.Statement{TableExpr: &clause.Expr{SQL: "users"}}}
				dt.Req(Request{
					Draw: 1,
					Columns: []ColumnRequest{
						{Name: "ID", Data: "id", Searchable: true, Orderable: true},
					},
				})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := New(nil)
			tt.setup(dt)

			err := dt.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error = %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSetTotalAndFilteredRecords(t *testing.T) {
	dt := New(nil)
	total, filtered := int64(100), int64(50)

	dt.SetTotalRecords(total).SetFilteredRecords(filtered)
	if *dt.totalRecords != total || *dt.filteredRecords != filtered {
		t.Errorf("expected totalRecords = %d, filteredRecords = %d, got %d, %d",
			total, filtered, *dt.totalRecords, *dt.filteredRecords)
	}
}

func TestSetRowAttributes(t *testing.T) {
	dt := New(nil)
	idFunc := func(row map[string]any) string { return row["id"].(string) }
	dataFunc := func(row map[string]any) map[string]any { return map[string]any{"custom": "data"} }

	dt.SetRowAttributes(idFunc, "row-class", dataFunc)
	if dt.rowIdFunc == nil || dt.rowClass != "row-class" || dt.rowDataFunc == nil {
		t.Errorf("row attributes were not set correctly")
	}
}

func TestDisableMethods(t *testing.T) {
	tests := []struct {
		name   string
		method func(*DataTable)
		field  string
		value  bool
	}{
		{
			name: "disable_search",
			method: func(dt *DataTable) {
				dt.DisableSearch()
			},
			field: "Searchable",
			value: false,
		},
		{
			name: "disable_order",
			method: func(dt *DataTable) {
				dt.DisableOrder()
			},
			field: "Orderable",
			value: false,
		},
		{
			name: "disable_pagination",
			method: func(dt *DataTable) {
				dt.DisablePagination()
			},
			field: "Paginate",
			value: false,
		},
		{
			name: "skip_paging",
			method: func(dt *DataTable) {
				dt.SkipPaging()
			},
			field: "Paginate",
			value: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := New(nil)
			tt.method(dt)

			v := reflect.ValueOf(dt.config).FieldByName(tt.field).Bool()
			if v != tt.value {
				t.Errorf("expected %s to be %v, got %v", tt.field, tt.value, v)
			}
		})
	}
}

func TestCaseInsensitive(t *testing.T) {
	dt := New(nil)

	dt.CaseInsensitive()
	if !dt.config.CaseInsensitive {
		t.Errorf("expected CaseInsensitive to be true, got false")
	}
}

func TestSetConfig(t *testing.T) {
	dt := New(nil)

	newConfig := Config{
		Searchable: true,
		Orderable:  false,
		Paginate:   true,
		Distinct:   false,
		GroupBy:    []string{"age"},
		Having:     []string{"COUNT(*) > 1"},
	}

	result := dt.SetConfig(newConfig)
	if !reflect.DeepEqual(dt.config, newConfig) {
		t.Errorf("expected config = %v, got %v", newConfig, dt.config)
	}

	if result != dt {
		t.Errorf("expected returned instance to be the same as the original DataTable instance")
	}
}
