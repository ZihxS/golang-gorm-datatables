package datatables

// applyCustomColumns applies all custom column editors to the given data.
//
// Custom column editors are functions that take a row (map[string]any) and
// return a new row with the same or different values. The editors are applied
// in the order they were added to the DataTable.
func (dt *DataTable) applyCustomColumns(data []map[string]any) {
	for _, editor := range dt.customCols {
		for i := range data {
			data[i] = editor(data[i])
		}
	}
}

// applyRowAttributes applies row-specific attributes to the given data.
//
// This function iterates through each row in the provided data slice and
// applies the row ID, class, and data attributes if they are defined.
// The row ID is determined by the rowIdFunc, which generates an ID based on
// each row's data. The row class is applied if the rowClass field is set.
// Additionally, custom data attributes are added to each row using the
// rowDataFunc, which returns a map of key-value pairs to be prefixed and
// appended as data-* attributes.
//
// The function modifies the data in place, enriching each row with the
// specified attributes.
func (dt *DataTable) applyRowAttributes(data []map[string]any) {
	for i := range data {
		row := data[i]
		if dt.rowIdFunc != nil {
			row[datatableRowID] = dt.rowIdFunc(row)
		}
		if dt.rowClass != "" {
			row[datatableRowClass] = dt.rowClass
		}
		if dt.rowDataFunc != nil {
			for k, v := range dt.rowDataFunc(row) {
				row[datatableRowDataPrefix+k] = v
			}
		}
	}
}

// getFilteredColumns returns a slice of columns that are whitelisted or
// blacklisted.
//
// If the selectedColumns slice is not empty, the function will return only
// the columns that are in the selectedColumns slice and are whitelisted.
// Otherwise, the function will return all columns that are whitelisted.
//
// The function filters the columns based on the whitelistColumns and
// blacklistColumns maps. If a column is in the whitelistColumns map, it is
// included in the result. If a column is in the blacklistColumns map, it is
// excluded from the result.
func (dt *DataTable) getFilteredColumns() []Column {
	filtered := []Column{}
	colMap := make(map[string]bool)
	if len(dt.selectedColumns) > 0 {
		for _, col := range dt.selectedColumns {
			colMap[col] = true
		}
		for _, col := range dt.columns {
			if colMap[col.Data] && dt.isColumnAllowed(col.Data) {
				filtered = append(filtered, col)
			}
		}
	} else {
		for _, col := range dt.columns {
			if dt.isColumnAllowed(col.Data) {
				filtered = append(filtered, col)
			}
		}
	}

	return filtered
}
