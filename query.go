package datatables

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// hasGroupByClause returns true if the query has a GROUP BY clause, false otherwise.
func hasGroupByClause(db *gorm.DB) bool {
	if _, exists := db.Statement.Clauses[queryGroupBy]; exists {
		return true
	}
	return false
}

// hasHavingClause returns true if the query has a HAVING clause, false otherwise.
func hasHavingClause(db *gorm.DB) bool {
	if _, exists := db.Statement.Clauses[queryHaving]; exists {
		return true
	}
	return false
}

// hasJoinClause returns true if the query has a JOIN clause, false otherwise.
func (dt *DataTable) hasJoinClause() bool {
	return len(dt.tx.Statement.Joins) > 0
}

// applyFilters applies the filters specified by the DataTable's Filters method
// to the query. Returns the updated query.
func (dt *DataTable) applyFilters(query *gorm.DB) *gorm.DB {
	for _, filter := range dt.filters {
		query = filter(query)
	}
	return query
}

// applyRelations applies the preloading of associations specified in the DataTable's
// relations slice to the query, but only if there are relations to preload and the
// query does not already have a JOIN clause. Returns the updated query.
func (dt *DataTable) applyRelations(query *gorm.DB) *gorm.DB {
	if len(dt.relations) > 0 && !dt.hasJoinClause() {
		query = query.Preload(strings.Join(dt.relations, ","))
	}
	return query
}

// applySearch applies search filtering to the query based on the DataTable's request configuration.
//
// The search is performed across all columns defined in the request that are allowed
// and marked as searchable. The search value can be either a plain text or a regex pattern,
// and case sensitivity is configurable. If the search value is empty or the search
// functionality is disabled, the query is returned unmodified. Returns the updated query.
func (dt *DataTable) applySearch(query *gorm.DB) *gorm.DB {
	if !dt.config.Searchable || dt.req.Search.Value == "" {
		return query
	}

	var conditions []clause.Expression
	for _, clientCol := range dt.req.Columns {
		if !dt.isColumnAllowed(clientCol.Data) {
			continue
		}
		if col, exists := dt.columnsMap[clientCol.Data]; exists && col.Searchable {
			val := dt.req.Search.Value
			if dt.config.CaseInsensitive {
				val = strings.ToLower(val)
			}
			if dt.req.Search.Regex {
				conditions = append(conditions, clause.Expr{
					SQL:  "? REGEXP ?",
					Vars: []any{clause.Column{Name: col.Name}, val},
				})
			} else {
				conditions = append(conditions, clause.Like{
					Column: clause.Column{Name: col.Name},
					Value:  "%" + val + "%",
				})
			}
		}
	}

	if len(conditions) > 0 {
		query = query.Where(clause.Or(conditions...))
	}

	return query
}

// executeQuery executes the given query and returns the result as a slice of
// maps, where each map represents a row in the result set.
//
// The function takes a gorm.DB query instance as an argument and executes the
// query using the Find method. The result is stored in the rawData variable,
// which is then returned to the caller along with any error that may have
// occurred. Returns the updated query.
func (dt *DataTable) executeQuery(query *gorm.DB) ([]map[string]any, error) {
	var rawData []map[string]any
	err := query.Find(&rawData).Error
	return rawData, err
}

// buildBaseQuery returns a gorm.DB query instance that is the base query used
// by the DataTable to generate the filtered, sorted, and paginated result set.
//
// The query is built by applying the relations specified by the DataTable's
// relations slice to the query, and then applying the filters specified by
// the DataTable's Filters method to the query. If the DataTable's model is a
// string, the query is built by using the Select method to select the columns
// specified by the DataTable's request configuration. Returns the updated query.
func (dt *DataTable) buildBaseQuery() *gorm.DB {
	var query *gorm.DB
	if _, ok := dt.model.(string); ok {
		query = dt.tx
		if dt.tx.Statement.Selects != nil {
			query = dt.tx.Select(
				strings.TrimSpace(
					strings.ReplaceAll(
						strings.Join(dt.tx.Statement.Selects, ", "),
						querySelect,
						"",
					),
				),
			)
		}
	} else {
		query = dt.tx.Model(dt.model)
	}
	query = dt.applyRelations(query)
	query = dt.applyFilters(query)
	return query
}

// buildCountQuery creates a new query session for counting records based on
// the provided baseQuery. If the DataTable configuration specifies Distinct
// as true, it applies a distinct selection on the "id" field, ensuring that
// only unique records are counted. Returns the modified query ready for
// counting the records.
func (dt *DataTable) buildCountQuery(baseQuery *gorm.DB) *gorm.DB {
	countQuery := baseQuery.Session(&gorm.Session{})

	if dt.config.Distinct {
		countQuery = countQuery.Distinct("id")
	}

	return countQuery
}

// buildFilteredQuery applies the search filter specified by the DataTable's
// request configuration to the provided base query. If the DataTable's
// configuration specifies GroupBy, it applies the specified group by clause
// to the query. If the query already has a group by clause, it replaces it
// with the new one. If the configuration specifies Having, it applies the
// specified having conditions to the query. If the query already has a having
// clause, it replaces it with the new one. Returns the updated query.
func (dt *DataTable) buildFilteredQuery(baseQuery *gorm.DB) *gorm.DB {
	query := baseQuery.Session(&gorm.Session{})
	query = dt.applySearch(query)

	if len(dt.config.GroupBy) > 0 {
		if !hasGroupByClause(query) {
			query = query.Group(strings.Join(dt.config.GroupBy, ", "))
		} else {
			delete(query.Statement.Clauses, queryGroupBy)
			query = query.Group(
				strings.TrimSpace(
					strings.ReplaceAll(
						strings.Join(dt.config.GroupBy, ", "),
						queryGroupBy,
						"",
					),
				),
			)
		}
		for _, cond := range dt.config.Having {
			if hasHavingClause(query) {
				delete(query.Statement.Clauses, queryHaving)
			}
			query = query.Having(
				strings.TrimSpace(
					strings.ReplaceAll(
						cond,
						queryHaving,
						"",
					),
				),
			)
		}
	}

	return query
}

// getTotalCount executes the count query and returns the total number of records
// in the table and any error that may have occurred. If the total number of records
// is already cached, it returns the cached value.
func (dt *DataTable) getTotalCount(countQuery *gorm.DB) (int64, error) {
	if dt.totalRecords != nil {
		return *dt.totalRecords, nil
	}

	if groupByClause, ok := countQuery.Statement.Clauses[queryGroupBy]; ok {
		expr, ok := groupByClause.Expression.(clause.GroupBy)
		if ok {
			newGroupBy := expr
			newGroupBy.Having = nil
			groupByClause.Expression = &newGroupBy
			countQuery.Statement.Clauses[queryGroupBy] = groupByClause
		}
	}

	var count int64
	err := countQuery.Count(&count).Error
	return count, err
}

// getFilteredCount executes the filtered query and returns the total number of records
// in the table that are visible after filtering and any error that may have occurred.
// If the total number of records is already cached, it returns the cached value.
// If the query has a GROUP BY clause, it executes a subquery to get the count.
func (dt *DataTable) getFilteredCount(filteredQuery *gorm.DB) (int64, error) {
	if dt.filteredRecords != nil {
		return *dt.filteredRecords, nil
	}

	var count int64

	if len(dt.config.GroupBy) > 0 {
		subQuery := filteredQuery.Session(&gorm.Session{})
		subQuery = dt.tx.Select(queryCount).Table("(?) subquery", subQuery)
		if dt.hasJoinClause() {
			subQuery.Statement.Joins = nil
		}
		delete(subQuery.Statement.Clauses, queryGroupBy)
		err := subQuery.Scan(&count).Error
		return count, err
	}

	err := filteredQuery.Count(&count).Error
	return count, err
}

// applyOrder applies the ordering specified by the DataTable's request configuration
// to the query. If ordering is disabled in the configuration, the query is returned
// unmodified. If the configuration specifies a union, it applies a default ordering
// by the "union_order" column. For each order in the request, it checks if the column
// is allowed and orderable, and applies the specified order direction. If no order
// is specified in the request, it applies the default sorting defined in the configuration.
// Returns the updated query with the applied order.
func (dt *DataTable) applyOrder(query *gorm.DB) *gorm.DB {
	if !dt.config.Orderable {
		return query
	}

	if dt.config.Union {
		return query.Order(clause.OrderByColumn{
			Column: clause.Column{Name: "union_order"},
			Desc:   false,
		})
	}

	for _, order := range dt.req.Order {
		if order.Column >= len(dt.req.Columns) {
			continue
		}
		clientCol := dt.req.Columns[order.Column]
		if !dt.isColumnAllowed(clientCol.Data) {
			continue
		}
		if col, exists := dt.columnsMap[clientCol.Data]; exists && col.Orderable {
			dir := strings.ToUpper(order.Dir)
			if dir != orderAscending && dir != orderDescending {
				dir = orderAscending
			}
			if col.Name != "" {
				query = query.Order(clause.OrderByColumn{
					Column: clause.Column{Name: col.Name},
					Desc:   strings.ToUpper(dir) == orderDescending,
				})
			}
		}
	}

	if len(dt.req.Order) == 0 && len(dt.config.DefaultSort) > 0 {
		for name, dir := range dt.config.DefaultSort {
			if col, exists := dt.columnsMap[name]; exists {
				colName := col.Name
				if colName == "" {
					colName = col.Data
				}
				if colName != "" {
					query = query.Order(clause.OrderByColumn{
						Column: clause.Column{Name: colName},
						Desc:   strings.ToUpper(dir) == orderDescending,
					})
				}
			}
		}
	}

	return query
}

// applyPagination applies pagination to the query if the DataTable's config
// has pagination enabled. Returns the updated query.
func (dt *DataTable) applyPagination(query *gorm.DB) *gorm.DB {
	if dt.config.Paginate {
		query = query.Offset(dt.req.Start).Limit(dt.req.Length)
	}
	return query
}

// checkComplexQuery inspects the DataTable's query to determine if it contains
// UNION, DISTINCT, GROUP BY, or HAVING clauses. It sets the appropriate flags
// in the DataTable's config field to indicate the presence of these clauses.
//
// Note that this function does not actually execute the query. It uses
// GORM's DryRun feature to generate the SQL without executing it.
func (dt *DataTable) checkComplexQuery() {
	var result []map[string]any
	tx := dt.tx.Session(&gorm.Session{DryRun: true}).Find(&result)

	sql := tx.Statement.SQL.String()
	sql = strings.ToUpper(sql)

	if strings.Contains(sql, queryUnion) {
		dt.config.Union = true
	}

	if strings.Contains(sql, queryDistinct) {
		dt.config.Distinct = true
	}

	if groupByIndex := strings.Index(sql, queryGroupBy); groupByIndex != -1 {
		endIndex := len(sql)
		if havingIndex := strings.Index(sql, queryHaving); havingIndex != -1 {
			endIndex = havingIndex
		}
		groupByClause := sql[groupByIndex:endIndex]
		dt.config.GroupBy = extractFields(groupByClause)
	}

	if havingIndex := strings.Index(sql, queryHaving); havingIndex != -1 {
		havingClause := sql[havingIndex:]
		dt.config.Having = extractFields(havingClause)
	}
}

// processQuery processes the DataTable's query by executing several steps to retrieve the data.
// It first checks for complex query clauses like UNION, DISTINCT, GROUP BY, and HAVING.
// Then, it builds the base query and creates a count and filtered query from it.
// The function retrieves the total record count and the filtered record count,
// applies ordering and pagination, and finally executes the query to get the data.
// Returns the raw data, total record count, filtered record count, and any error encountered.
func (dt *DataTable) processQuery() (any, int64, int64, error) {
	dt.checkComplexQuery()
	baseQuery := dt.buildBaseQuery()
	countQuery := dt.buildCountQuery(baseQuery)
	filteredQuery := dt.buildFilteredQuery(baseQuery)

	total, err := dt.getTotalCount(countQuery)
	if err != nil {
		return nil, 0, 0, err
	}

	filtered, err := dt.getFilteredCount(filteredQuery)
	if err != nil {
		return nil, 0, 0, err
	}

	query := dt.applyOrder(filteredQuery)
	query = dt.applyPagination(query)
	rawData, err := dt.executeQuery(query)
	if err != nil {
		return nil, 0, 0, err
	}

	return rawData, total, filtered, nil
}

// Raw returns the raw data retrieved from the database by executing the DataTable's query.
//
// This function does not apply any custom column rendering functions or row attributes.
// It returns the raw data as retrieved from the database, along with any error that may have occurred.
func (dt *DataTable) Raw() (any, error) {
	data, _, _, err := dt.processQuery()
	return data, err
}
