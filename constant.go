package datatables

// Constants for specifying order direction in the DataTable API.
const (
	orderAscending  = "ASC"  // Sort in ascending order.
	orderDescending = "DESC" // Sort in descending order.
)

// Constants representing SQL query clauses used in DataTable processing.
const (
	querySelect   = "SELECT"            // SQL SELECT clause.
	queryUnion    = "UNION"             // SQL UNION operator.
	queryDistinct = "DISTINCT"          // SQL DISTINCT keyword.
	queryGroupBy  = "GROUP BY"          // SQL GROUP BY clause.
	queryHaving   = "HAVING"            // SQL HAVING clause.
	queryCount    = "COUNT(*) AS count" // SQL COUNT function with alias.
)

// Constants used by DataTables in the JSON response to represent the row
// attributes.
//
// These constants are used by the DataTables library to represent the row
// attributes in the JSON response.
//
// The DT_RowId represents the row ID attribute.
// The DT_RowClass represents the row class attribute.
// The DT_RowData_ prefix represents the row data attribute.
const (
	datatableRowID         = "DT_RowId"    // Row ID attribute.
	datatableRowClass      = "DT_RowClass" // Row class attribute.
	datatableRowDataPrefix = "DT_RowData_" // Row data attribute prefix.
)
