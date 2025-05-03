package datatables

// Config holds the configuration options for a DataTable.
//
// The Config struct allows customization of various features
// such as searchability, ordering, pagination, and more.
//
// Fields:
//   - Searchable: Enables or disables global searching.
//   - Orderable: Enables or disables column ordering.
//   - Paginate: Enables or disables pagination.
//   - Union: Allows the use of UNION in queries.
//   - Distinct: Enables DISTINCT selection in queries.
//   - CaseInsensitive: Enables case-insensitive searches.
//   - ResponseFormat: Specifies the format of the response.
//   - GroupBy: Specifies columns for GROUP BY clause.
//   - Having: Specifies conditions for HAVING clause.
//   - DefaultSort: Specifies default sorting for columns.
type Config struct {
	Searchable      bool
	Orderable       bool
	Paginate        bool
	Union           bool
	Distinct        bool
	CaseInsensitive bool
	ResponseFormat  string
	GroupBy         []string
	Having          []string
	DefaultSort     map[string]string
}
