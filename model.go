package datatables

import (
	"errors"
	"regexp"

	"gorm.io/gorm"
)

// DataTable represents the configuration and data for a datatables request.
//
// It contains the configuration, request, and data for a datatables request.
// The configuration is represented by the Config field, which is used to
// configure the DataTable.

// The request is represented by the Request field, which is used to specify
// the request configuration.

// The data is represented by the totalRecords, filteredRecords, and columns
// fields, which are used to store the data.
type DataTable struct {
	totalRecords     *int64
	filteredRecords  *int64
	rowClass         string
	model            any
	tx               *gorm.DB
	req              Request
	config           Config
	relations        []string
	selectedColumns  []string
	columns          []Column
	whitelistColumns map[string]bool
	blacklistColumns map[string]bool
	additionalData   map[string]any
	columnsMap       map[string]Column
	rowIdFunc        func(map[string]any) string
	rowDataFunc      func(map[string]any) map[string]any
	filters          []func(*gorm.DB) *gorm.DB
	customCols       []func(map[string]any) map[string]any
}

// Model sets the model to be used for the datatables request.
//
// This can either be a struct or a string representing the table name.
//
// If a struct is provided, it should have the same fields as the table
// being queried. If a string is provided, it should be the table name.
// The query will be executed on this table.
//
// The model can also be set when creating a new DataTable with the
// New function.
func (dt *DataTable) Model(model any) *DataTable {
	dt.model = model
	return dt
}

// Req sets the request parameters for the DataTable and adds the specified columns.
//
// The request contains various configurations such as draw counter, pagination info,
// search terms, and column specifications. Each column specified in the request is
// added to the DataTable. The columns are set with their respective properties,
// including name, data, searchable, and orderable attributes.
//
// Returns the updated DataTable instance.
func (dt *DataTable) Req(req Request) *DataTable {
	dt.req = req
	for _, v := range dt.req.Columns {
		dt = dt.AddColumn(Column{
			Name:       v.Name,
			Data:       v.Data,
			Searchable: v.Searchable,
			Orderable:  v.Orderable,
			RenderFunc: nil,
		})
	}
	return dt
}

// Only sets the selectedColumns field of the DataTable to the specified columns.
//
// This function allows the user to specify which columns should be included
// in the DataTable's operations. It takes one or more string arguments,
// representing the Data fields of the columns to be included. The selected
// columns will be used in subsequent operations, such as filtering and
// rendering the table. The function returns the updated DataTable instance.
func (dt *DataTable) Only(columns ...string) *DataTable {
	dt.selectedColumns = columns
	return dt
}

// With appends the specified relations to the DataTable's relations slice.
//
// This function allows the user to specify related models that should be
// included in the query. The relations are typically defined as strings
// representing the names of the related models. These relations will be
// processed during query execution to preload associated data.
//
// Returns the updated DataTable instance.
func (dt *DataTable) With(relations ...string) *DataTable {
	dt.relations = append(dt.relations, relations...)
	return dt
}

// WithData adds a key-value pair to the DataTable's additional data map.
//
// This function allows the user to specify arbitrary key-value pairs that
// should be included in the DataTable's response. The key should be a string
// representing the key of the value, and the value should be the value
// itself. The function returns the updated DataTable instance.
func (dt *DataTable) WithData(key string, value any) *DataTable {
	dt.additionalData[key] = value
	return dt
}

// WithNumber adds a column named "No" to the DataTable, which is non-searchable
// and non-orderable. The column is then blacklisted, meaning it will not be
// included in the final response. This function returns the updated DataTable
// instance.
func (dt *DataTable) WithNumber() *DataTable {
	dt.AddColumn(Column{Name: "No", Data: "no", Searchable: false, Orderable: false, RenderFunc: nil})
	dt.BlacklistColumn("no")
	return dt
}

// Filter adds the specified filter function to the DataTable's filters slice.
//
// This function allows the user to specify custom filtering logic that should
// be applied to the DataTable's query. The function takes a single argument, a
// function that takes a gorm.DB instance and returns a gorm.DB instance. The
// function should apply its desired filtering logic to the provided gorm.DB
// instance and return the updated instance.
//
// Returns the updated DataTable instance.
func (dt *DataTable) Filter(filterFunc func(*gorm.DB) *gorm.DB) *DataTable {
	dt.filters = append(dt.filters, filterFunc)
	return dt
}

// Validate checks the integrity of the DataTable configuration and request.
//
// It ensures that either a model or a transaction (tx) with a valid gorm statement
// is provided. If a model is not explicitly set, it attempts to derive it from the
// gorm statement. The function also validates the request by checking the draw and
// columns parameters. If a regex search pattern is provided, it verifies that the
// pattern is valid. Returns an error if any of these validations fail, otherwise
// returns nil.
func (dt *DataTable) Validate() error {
	if dt.model == nil {
		if dt.tx == nil {
			return errors.New("no tx or model provided")
		}
		if dt.tx.Statement == nil {
			return errors.New("gorm statement is required")
		}
		if dt.tx.Statement.Model == nil {
			if dt.tx.Statement.TableExpr == nil || dt.tx.Statement.TableExpr.SQL == "" {
				return errors.New("model is required")
			}
			dt.model = dt.tx.Statement.TableExpr.SQL
			goto afterModel
		}
		dt.model = dt.tx.Statement.Model
	}

afterModel:
	if dt.req.Draw == 0 && len(dt.req.Columns) == 0 {
		return errors.New("invalid request")
	}

	if dt.req.Search.Regex {
		if _, err := regexp.Compile(dt.req.Search.Value); err != nil {
			return errors.New("invalid regex search pattern")
		}
	}

	return nil
}

// SetTotalRecords sets the total number of records in the table.
//
// This is a convenience method, and is used internally by the DataTable
// to store the total number of records returned by the count query.
//
// Returns the updated DataTable instance.
func (dt *DataTable) SetTotalRecords(count int64) *DataTable {
	dt.totalRecords = &count
	return dt
}

// SetFilteredRecords sets the total number of records in the table that
// are visible after filtering.
//
// This is a convenience method, and is used internally by the DataTable
// to store the total number of records returned by the filtered query.
//
// Returns the updated DataTable instance.
func (dt *DataTable) SetFilteredRecords(count int64) *DataTable {
	dt.filteredRecords = &count
	return dt
}

// SetRowAttributes sets the row attributes of the DataTable.
//
// This method is a convenience method that can be used to set the row
// attributes of the DataTable. The row attributes are used by the
// DataTable to generate the HTML for the table.
//
// The idFunc parameter is a function that takes a row and returns the ID
// of the row. The class parameter is the class to be applied to the
// table row. The dataFunc parameter is a function that takes a row and
// returns a map of data to be added to the table row as data-* attributes.
//
// Returns the updated DataTable instance.
func (dt *DataTable) SetRowAttributes(idFunc func(map[string]any) string, class string, dataFunc func(map[string]any) map[string]any) *DataTable {
	dt.rowIdFunc = idFunc
	dt.rowClass = class
	dt.rowDataFunc = dataFunc
	return dt
}

// SetConfig sets the DataTable's configuration to the specified Config
// instance.
//
// This method is a convenience method that can be used to set the DataTable's
// configuration after it has been created. The Config instance should contain
// the desired settings for the DataTable, such as whether or not to allow
// searching, ordering, and pagination. The Config instance should also contain
// the desired settings for the DataTable's request, such as the draw counter,
// start, and length.
//
// Returns the updated DataTable instance.
func (dt *DataTable) SetConfig(config Config) *DataTable {
	dt.config = config
	return dt
}

// DisableSearch disables the search functionality for the DataTable.
//
// This method sets the Searchable field in the DataTable's configuration to false,
// effectively disabling any search operations on the DataTable. After calling this
// method, the DataTable will not perform search filtering on the data.
//
// Returns the updated DataTable instance.
func (dt *DataTable) DisableSearch() *DataTable {
	dt.config.Searchable = false
	return dt
}

// DisableOrder disables the ordering functionality for the DataTable.
//
// This method sets the Orderable field in the DataTable's configuration to false,
// effectively disabling any ordering operations on the DataTable. After calling this
// method, the DataTable will not perform ordering on the data.
//
// Returns the updated DataTable instance.
func (dt *DataTable) DisableOrder() *DataTable {
	dt.config.Orderable = false
	return dt
}

// DisablePagination disables the pagination functionality for the DataTable.
//
// This method sets the Paginate field in the DataTable's configuration to false,
// effectively disabling any pagination operations on the DataTable. After calling
// this method, the DataTable will not perform pagination on the data.
//
// Returns the updated DataTable instance.
func (dt *DataTable) DisablePagination() *DataTable {
	dt.config.Paginate = false
	return dt
}

// SkipPaging disables the pagination functionality for the DataTable.
//
// This is a convenience method, and is equivalent to calling
// DisablePagination. After calling this method, the DataTable will not
// perform pagination on the data.
//
// Returns the updated DataTable instance.
func (dt *DataTable) SkipPaging() *DataTable {
	return dt.DisablePagination()
}

// CaseInsensitive enables case-insensitive search for the DataTable.
//
// This method sets the CaseInsensitive field in the DataTable's configuration to true,
// allowing the DataTable to perform case-insensitive search operations. After calling
// this method, any search queries executed by the DataTable will ignore the case
// of the search terms.
//
// Returns the updated DataTable instance.
func (dt *DataTable) CaseInsensitive() *DataTable {
	dt.config.CaseInsensitive = true
	return dt
}
