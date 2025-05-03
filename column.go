package datatables

import (
	"slices"
)

// Column represents a single column in a DataTable.
//
// A Column has the following properties:
//
// Fields:
//   - Searchable: A boolean indicating whether the column is searchable.
//   - Orderable: A boolean indicating whether the column is orderable.
//   - Name: The display name of the column.
//   - Data: The data property name of the column.
//   - RenderFunc: An optional function that can be used to render the column value.
type Column struct {
	Searchable bool
	Orderable  bool
	Name       string
	Data       string
	RenderFunc func(map[string]any) any
}

// initColumnsMap initializes the columnsMap field of DataTable with the
// columns that were passed to it. It iterates over the columns slice and
// adds each column to the columnsMap with its Data field as the key.
//
// This function is called automatically when a DataTable is initialized.
func (dt *DataTable) initColumnsMap() {
	dt.columnsMap = make(map[string]Column)
	for _, col := range dt.columns {
		dt.columnsMap[col.Data] = col
	}
}

// isColumnAllowed checks if a column with the given name is allowed based on the
// whitelist and blacklist constraints. If both whitelist and blacklist are empty,
// all columns are allowed. If the whitelist is non-empty, only columns explicitly
// listed are allowed. If the blacklist is non-empty and the whitelist is empty,
// only columns not listed in the blacklist are allowed.
func (dt *DataTable) isColumnAllowed(name string) bool {
	if len(dt.whitelistColumns) == 0 && len(dt.blacklistColumns) == 0 {
		return true
	}

	if len(dt.whitelistColumns) > 0 {
		return dt.whitelistColumns[name]
	}

	return !dt.blacklistColumns[name]
}

// AddColumn adds a column to the DataTable. If a column with the same Data
// field exists, it is overwritten. The column is added to the columnsMap
// with the Data field as the key.
func (dt *DataTable) AddColumn(col Column) *DataTable {
	if _, ok := dt.columnsMap[col.Data]; !ok {
		dt.columns = append(dt.columns, col)
	}
	dt.columnsMap[col.Data] = col
	return dt
}

// AddColumns adds multiple columns to the DataTable. If a column with the same
// Data field exists, it is overwritten. The columns are added to the columnsMap
// with the Data field as the key.
func (dt *DataTable) AddColumns(columns ...Column) *DataTable {
	for _, v := range columns {
		newCol := Column{
			Name:       v.Name,
			Data:       v.Data,
			Searchable: v.Searchable,
			Orderable:  v.Orderable,
			RenderFunc: v.RenderFunc,
		}
		dt.AddColumn(newCol)
	}
	return dt
}

// EditColumn edits the render function of a column with the given name.
//
// If a column with the given name exists, the render function is replaced with a
// new one that calls the given editFunc with the value of the column from the
// given row. If the column does not exist, the function does nothing.
//
// The RenderFunc field of the column is replaced with a new one, and the new
// column is stored in the columnsMap with the Data field as the key.
func (dt *DataTable) EditColumn(name string, editFunc func(any) any) *DataTable {
	if col, exists := dt.columnsMap[name]; exists {
		col.RenderFunc = func(row map[string]any) any {
			value := row[col.Data]
			return editFunc(value)
		}
		dt.columnsMap[name] = col
	}
	return dt
}

// RemoveColumn removes one or more columns from the DataTable. The columns are
// removed from the selectedColumns and columns fields of the DataTable. If the
// selectedColumns field is empty, the columns are removed from the columns
// field. If the selectedColumns field is not empty, the columns are removed from
// the selectedColumns field and the columns field is updated accordingly.
//
// The function takes one or more string arguments, which are the Data fields of
// the columns to be removed. If a column with one of the given Data fields
// exists, it is removed from the DataTable. If a column does not exist, the
// function does nothing.
func (dt *DataTable) RemoveColumn(data ...string) *DataTable {
	exclude := make(map[string]bool)
	for _, d := range data {
		exclude[d] = true
	}

	if len(dt.selectedColumns) == 0 && dt.columns != nil && len(dt.columns) > 0 {
		var filtered []Column
		for _, col := range dt.columns {
			if !exclude[col.Data] {
				filtered = append(filtered, col)
			}
		}
		dt.columns = filtered
		dt.selectedColumns = make([]string, 0)
		for _, c := range dt.columns {
			dt.selectedColumns = append(dt.selectedColumns, c.Data)
		}
	} else {
		var filtered []string
		for _, d := range dt.selectedColumns {
			if !exclude[d] {
				filtered = append(filtered, d)
			}
		}
		dt.selectedColumns = filtered
		dt.columns = make([]Column, 0)
		for _, d := range dt.selectedColumns {
			dt.columns = append(dt.columns, dt.columnsMap[d])
		}
	}

	return dt
}

// WhitelistColumn marks one or more columns as whitelisted. Only columns that are
// whitelisted will be included in the final response. If no columns are passed,
// this function does nothing.
func (dt *DataTable) WhitelistColumn(columns ...string) *DataTable {
	for _, col := range columns {
		dt.whitelistColumns[col] = true
	}
	return dt
}

// BlacklistColumn marks one or more columns as blacklisted. Columns that are
// blacklisted will be excluded from the final response. If no columns are passed,
// this function does nothing.
func (dt *DataTable) BlacklistColumn(columns ...string) *DataTable {
	for _, col := range columns {
		dt.blacklistColumns[col] = true
	}
	return dt
}

// FinalizeResponseColumns removes any columns from the data that are not
// whitelisted or that are blacklisted.
func (dt *DataTable) FinalizeResponseColumns(data []map[string]any) []map[string]any {
	for _, row := range data {
		for keyCol := range row {
			if !slices.Contains(dt.selectedColumns, keyCol) {
				delete(row, keyCol)
			}
		}
	}
	return data
}
