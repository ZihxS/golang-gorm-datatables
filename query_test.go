package datatables

import (
	"database/sql/driver"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Profile struct {
	ID      int
	UserID  int
	Details string
}

type User struct {
	ID      int
	Name    string
	Profile []Profile `gorm:"foreignKey:UserID"`
}

func TestApplyFilters(t *testing.T) {
	tests := []struct {
		name         string
		filters      []func(*gorm.DB) *gorm.DB
		query        string
		args         []driver.Value
		expectedRows *sqlmock.Rows
	}{
		{
			name:    "no_filters",
			query:   "SELECT * FROM `users`",
			args:    nil,
			filters: nil,
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).
				AddRow(1, "John Doe", 25),
		},
		{
			name: "single_filter",
			filters: []func(*gorm.DB) *gorm.DB{
				func(query *gorm.DB) *gorm.DB {
					return query.Where("age > ?", 18)
				},
			},
			query: "SELECT * FROM `users` WHERE age > ?",
			args:  []driver.Value{18},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).
				AddRow(1, "John Doe", 25),
		},
		{
			name: "multiple_filters",
			filters: []func(*gorm.DB) *gorm.DB{
				func(query *gorm.DB) *gorm.DB {
					return query.Where("age > ?", 18)
				},
				func(query *gorm.DB) *gorm.DB {
					return query.Where("name LIKE ?", "%John%")
				},
			},
			query: "SELECT * FROM `users` WHERE age > ? AND name LIKE ?",
			args:  []driver.Value{18, "%John%"},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).
				AddRow(1, "John Doe", 25),
		},
	}

	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery(qm(tt.query)).WithArgs(tt.args...).
				WillReturnRows(tt.expectedRows)

			dt := New(db)
			dt.filters = tt.filters
			query := dt.tx.Model(&User{})
			result := dt.applyFilters(query)

			var users []User
			if err := result.Find(&users).Error; err != nil {
				t.Fatalf("failed to execute query: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestApplyRelations(t *testing.T) {
	tests := []struct {
		name         string
		relations    []string
		query        string
		args         []driver.Value
		expectedRows *sqlmock.Rows
	}{
		{
			name:      "no_relation",
			query:     "SELECT * FROM `users`",
			args:      nil,
			relations: nil,
			expectedRows: sqlmock.NewRows([]string{"id", "name"}).
				AddRow(1, "ZihxS"),
		},
		{
			name:      "relation",
			query:     "SELECT * FROM `profiles` WHERE `profiles`.`user_id` = ?",
			args:      []driver.Value{1},
			relations: []string{"Profile"},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "user_id"}).
				AddRow(1, "ZihxS", 1),
		},
	}

	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "relation" {
				mock.ExpectQuery(qm("SELECT * FROM `users`")).
					WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
						AddRow(1, "ZihxS"))
			}
			mock.ExpectQuery(qm(tt.query)).WithArgs(tt.args...).
				WillReturnRows(tt.expectedRows)

			dt := New(db)
			if len(tt.relations) > 0 {
				dt.With(tt.relations...)
			}
			query := dt.tx.Model(&User{})
			result := dt.applyRelations(query)

			var users []User
			if err := result.Find(&users).Error; err != nil {
				t.Fatalf("failed to execute query: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestApplySearch(t *testing.T) {
	tests := []struct {
		name         string
		searchable   bool
		searchValue  string
		searchRegex  bool
		query        string
		args         []driver.Value
		expectedRows *sqlmock.Rows
	}{
		{
			name:         "no_search",
			searchable:   false,
			query:        "SELECT * FROM `users`",
			args:         nil,
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "John Doe", 25),
		},
		{
			name:         "single_column_like_search",
			searchable:   true,
			searchValue:  "John",
			searchRegex:  false,
			query:        "SELECT * FROM `users` WHERE (`name` LIKE ? OR `age` LIKE ?)",
			args:         []driver.Value{"%john%", "%john%"},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "John Doe", 25),
		},
		{
			name:         "regex_search",
			searchable:   true,
			searchValue:  "J.*n",
			searchRegex:  true,
			query:        "SELECT * FROM `users` WHERE (`name` REGEXP ? OR `age` REGEXP ?)",
			args:         []driver.Value{"j.*n", "j.*n"},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "John Doe", 25),
		},
		{
			name:         "whitelist_column",
			searchable:   true,
			searchValue:  "john",
			searchRegex:  false,
			query:        "SELECT * FROM `users` WHERE `name` LIKE ?",
			args:         []driver.Value{"%john%"},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "John Doe", 25),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbMock, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer dbMock.Close()

			dialector := mysql.New(mysql.Config{
				Conn:                      dbMock,
				SkipInitializeWithVersion: true,
			})
			db, err := gorm.Open(dialector, &gorm.Config{})
			if err != nil {
				t.Fatalf("failed to open gorm DB: %v", err)
			}

			mock.ExpectQuery(qm(tt.query)).WithArgs(tt.args...).
				WillReturnRows(tt.expectedRows)

			dt := &DataTable{
				tx: db,
				config: Config{
					Searchable:      true,
					CaseInsensitive: true,
				},
				req: Request{
					Search: Search{
						Value: "John",
						Regex: false,
					},
					Columns: []ColumnRequest{
						{Data: "name", Searchable: true},
						{Data: "age", Searchable: true},
					},
				},
				columnsMap: map[string]Column{
					"name": {Name: "name", Searchable: true},
					"age":  {Name: "age", Searchable: true},
				},
			}
			dt.config.Searchable = tt.searchable
			dt.req.Search = Search{Value: tt.searchValue, Regex: tt.searchRegex}
			dt.columnsMap = map[string]Column{
				"name": {Name: "name", Searchable: true},
				"age":  {Name: "age", Searchable: true},
			}

			if tt.name == "whitelist_column" {
				dt.whitelistColumns = map[string]bool{"name": true}
			}

			query := dt.tx.Model(&User{})
			result := dt.applySearch(query)

			var users []User
			if err := result.Find(&users).Error; err != nil {
				t.Fatalf("failed to execute query: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestExecuteQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		args         []driver.Value
		expectedRows *sqlmock.Rows
		expectedErr  error
	}{
		{
			name:  "query_success",
			query: "SELECT * FROM `users`",
			args:  nil,
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).
				AddRow(1, "John Doe", 25).
				AddRow(2, "Jane Smith", 30),
			expectedErr: nil,
		},
		{
			name:         "query_failure",
			query:        "SELECT * FROM `users`",
			args:         nil,
			expectedRows: nil,
			expectedErr:  gorm.ErrInvalidData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbMock, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer dbMock.Close()

			dialector := mysql.New(mysql.Config{
				Conn:                      dbMock,
				SkipInitializeWithVersion: true,
			})
			db, err := gorm.Open(dialector, &gorm.Config{})
			if err != nil {
				t.Fatalf("failed to open gorm DB: %v", err)
			}

			if tt.expectedErr != nil {
				mock.ExpectQuery(qm(tt.query)).
					WillReturnError(tt.expectedErr)
			} else {
				mock.ExpectQuery(qm(tt.query)).
					WithArgs(tt.args...).
					WillReturnRows(tt.expectedRows)
			}

			dt := New(db)
			query := dt.tx.Model(&User{})
			result, err := dt.executeQuery(query)

			if err != tt.expectedErr {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}

			if tt.expectedErr == nil {
				expected := []map[string]any{
					{"id": 1, "name": "John Doe", "age": 25},
					{"id": 2, "name": "Jane Smith", "age": 30},
				}
				if !reflect.DeepEqual(normalizeResponse(result), normalizeResponse(expected)) {
					t.Errorf("expected %v, got %v", expected, result)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestBuildBaseQuery(t *testing.T) {
	tests := []struct {
		name         string
		relations    []string
		filters      []func(*gorm.DB) *gorm.DB
		query        string
		args         []driver.Value
		expectedRows *sqlmock.Rows
	}{
		{
			name:         "no_relations_or_filters",
			query:        "SELECT * FROM `users`",
			args:         nil,
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "John Doe", 25),
		},
		{
			name:         "with_relations_only",
			relations:    []string{"Profile"},
			query:        "SELECT * FROM `profiles` WHERE `profiles`.`user_id` = ?",
			args:         []driver.Value{1},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "user_id"}).AddRow(1, "John Doe", 1),
		},
		{
			name: "with_filters_only",
			filters: []func(*gorm.DB) *gorm.DB{
				func(query *gorm.DB) *gorm.DB { return query.Where("age > ?", 18) },
			},
			query:        "SELECT * FROM `users` WHERE age > ?",
			args:         []driver.Value{18},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "John Doe", 25),
		},
		{
			name: "model_string",
			filters: []func(*gorm.DB) *gorm.DB{
				func(query *gorm.DB) *gorm.DB { return query.Where("age > ?", 18) },
			},
			query:        "SELECT * FROM `users` WHERE age > ?",
			args:         []driver.Value{18},
			expectedRows: sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "John Doe", 25),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbMock, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer dbMock.Close()

			dialector := mysql.New(mysql.Config{
				Conn:                      dbMock,
				SkipInitializeWithVersion: true,
			})
			db, err := gorm.Open(dialector, &gorm.Config{})
			if err != nil {
				t.Fatalf("failed to open gorm DB: %v", err)
			}

			dt := New(db)
			dt.filters = tt.filters

			if len(tt.relations) > 0 {
				dt.With(tt.relations...)
				mock.ExpectQuery(qm("SELECT * FROM `users`")).
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).
						AddRow(1, "John Doe", 1))
			}
			mock.ExpectQuery(qm(tt.query)).
				WithArgs(tt.args...).
				WillReturnRows(tt.expectedRows)

			if tt.name == "model_string" {
				dt.model = "users"
				dt.tx.Statement.Selects = []string{"*"}
			}
			query := dt.buildBaseQuery()

			var users []User
			if err := query.Find(&users).Error; err != nil {
				t.Fatalf("failed to execute query: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestBuildCountQuery(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	dt := New(db)

	tests := []struct {
		name          string
		config        Config
		expectedQuery string
		expectedCount int64
	}{
		{
			name: "without_distinct",
			config: Config{
				Distinct: false,
			},
			expectedQuery: "SELECT count(*) FROM `users`",
			expectedCount: 25,
		},
		{
			name: "with_distinct",
			config: Config{
				Distinct: true,
			},
			expectedQuery: "SELECT COUNT(DISTINCT(`id`)) FROM `users`",
			expectedCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt.config = tt.config

			mock.ExpectQuery(qm(tt.expectedQuery)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tt.expectedCount))

			baseQuery := dt.tx.Model(&User{})

			countQuery := dt.buildCountQuery(baseQuery)
			var count int64
			if err := countQuery.Count(&count).Error; err != nil {
				t.Fatalf("failed to execute count query: %v", err)
			}

			if count != tt.expectedCount {
				t.Errorf("expected count %d, got %d", tt.expectedCount, count)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestBuildFilteredQuery(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	tests := []struct {
		name           string
		config         Config
		searchFilter   string
		expectedQuery  string
		expectedArgs   []driver.Value
		expectedResult []map[string]any
	}{
		{
			name: "with_group_by_and_having",
			config: Config{
				Searchable: true,
				Orderable:  true,
				Paginate:   true,
				GroupBy:    []string{"group"},
				Having:     []string{"COUNT(*) > 1"},
			},
			searchFilter:  "`name` LIKE ?",
			expectedQuery: "SELECT * FROM `users` WHERE `name` LIKE ? GROUP BY `group` HAVING COUNT(*) > 1",
			expectedArgs:  []driver.Value{"%John%"},
			expectedResult: []map[string]any{
				{"id": 1, "name": "John Doe", "age": 25, "group": "A"},
				{"id": 2, "name": "John Smith", "age": 30, "group": "B"},
			},
		},
		{
			name: "without_group_by_and_having",
			config: Config{
				Searchable: true,
				Orderable:  true,
				Paginate:   true,
			},
			searchFilter:  "`name` LIKE ?",
			expectedQuery: "SELECT * FROM `users` WHERE `name` LIKE ?",
			expectedArgs:  []driver.Value{"%John%"},
			expectedResult: []map[string]any{
				{"id": 1, "name": "John Doe", "age": 25, "group": "A"},
				{"id": 2, "name": "John Smith", "age": 30, "group": "B"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery(qm(tt.expectedQuery)).
				WithArgs(tt.expectedArgs...).
				WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age", "group"}).
					AddRow(1, "John Doe", 25, "A").
					AddRow(2, "John Smith", 30, "B"))

			dt := New(db)
			dt.config = tt.config

			baseQuery := dt.tx.Model(&User{}).Where(tt.searchFilter, "%John%")

			if len(dt.config.Having) > 0 {
				baseQuery.Statement.Clauses[queryHaving] = clause.Clause{}
			}

			if len(dt.config.GroupBy) > 0 {
				baseQuery.Statement.Clauses[queryGroupBy] = clause.Clause{}
			}

			filteredQuery := dt.buildFilteredQuery(baseQuery)

			var results []map[string]any
			err := filteredQuery.Find(&results).Error

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			normalizedResponse := normalizeResponse(results)
			normalizedExpected := normalizeResponse(tt.expectedResult)

			if !reflect.DeepEqual(normalizedResponse, normalizedExpected) {
				t.Errorf("expected results = %v, got %v", normalizedExpected, normalizedResponse)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestGetTotalCount(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	dt := New(db)

	tests := []struct {
		name          string
		cachedCount   *int64
		mockQuery     string
		mockArgs      []driver.Value
		mockRows      *sqlmock.Rows
		expectedCount int64
		expectedError error
	}{
		{
			name:          "with_cached_total_records",
			cachedCount:   &[]int64{42}[0],
			expectedCount: 42,
		},
		{
			name:          "with_query_execution",
			mockQuery:     "SELECT count(*) FROM `users`",
			mockRows:      sqlmock.NewRows([]string{"count"}).AddRow(25),
			expectedCount: 25,
		},
		{
			name:          "with_query_failure",
			mockQuery:     "SELECT count(*) FROM `users`",
			mockRows:      sqlmock.NewRows([]string{"count"}).AddRow(nil),
			expectedCount: 0,
			expectedError: gorm.ErrInvalidData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt.totalRecords = tt.cachedCount

			if tt.mockQuery != "" {
				mock.ExpectQuery(qm(tt.mockQuery)).
					WillReturnRows(tt.mockRows).
					WillReturnError(tt.expectedError)
			}

			countQuery := dt.tx.Model(&User{})
			count, err := dt.getTotalCount(countQuery)

			if err != tt.expectedError {
				t.Fatalf("expected error %v, got %v", tt.expectedError, err)
			}

			if count != tt.expectedCount {
				t.Errorf("expected count %d, got %d", tt.expectedCount, count)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestGetFilteredCount(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	dt := New(db)

	tests := []struct {
		name          string
		cachedCount   *int64
		groupBy       []string
		mockQuery     string
		mockArgs      []driver.Value
		mockRows      *sqlmock.Rows
		expectedCount int64
		expectedError error
	}{
		{
			name:          "with_cached_filtered_records",
			cachedCount:   &[]int64{42}[0],
			expectedCount: 42,
		},
		{
			name:          "without_group_by",
			mockQuery:     "SELECT count(*) FROM `users`",
			mockRows:      sqlmock.NewRows([]string{"count"}).AddRow(25),
			expectedCount: 25,
		},
		{
			name:          "with_group_by",
			groupBy:       []string{"age"},
			mockQuery:     "SELECT COUNT(*) AS count FROM (SELECT * FROM `users` GROUP BY `age`) subquery",
			mockRows:      sqlmock.NewRows([]string{"count"}).AddRow(10),
			expectedCount: 10,
		},
		{
			name:          "query_failure",
			mockQuery:     "SELECT count(*) FROM `users`",
			mockRows:      sqlmock.NewRows([]string{"count"}).AddRow(25),
			expectedCount: 0,
			expectedError: gorm.ErrInvalidData,
		},
		{
			name:          "with_group_by_join",
			groupBy:       []string{"age"},
			mockQuery:     "SELECT COUNT(*) AS count FROM (SELECT `users`.`id`,`users`.`name` FROM `users` INNER JOIN `profiles` ON `users`.`id` = `profiles`.`user_id` GROUP BY `age`) subquery",
			mockRows:      sqlmock.NewRows([]string{"count"}).AddRow(10),
			expectedCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt.filteredRecords = tt.cachedCount
			dt.config.GroupBy = tt.groupBy

			if tt.mockQuery != "" {
				if tt.expectedError != nil {
					mock.ExpectQuery(qm(tt.mockQuery)).
						WillReturnError(gorm.ErrInvalidData)
				} else {
					mock.ExpectQuery(qm(tt.mockQuery)).
						WillReturnRows(tt.mockRows)
				}
			}

			filteredQuery := dt.tx.Model(&User{})
			filteredQuery = dt.buildFilteredQuery(filteredQuery)

			if tt.name == "with_group_by_join" {
				filteredQuery = filteredQuery.Joins("INNER JOIN `profiles` ON `users`.`id` = `profiles`.`user_id`")
			}

			count, err := dt.getFilteredCount(filteredQuery)

			if err != tt.expectedError {
				t.Fatalf("expected error %v, got %v", tt.expectedError, err)
			}

			if count != tt.expectedCount {
				t.Errorf("expected count %d, got %d", tt.expectedCount, count)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestApplyOrder(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	dt := New(db)
	dt.columnsMap = map[string]Column{
		"name": {Name: "name", Orderable: true},
		"age":  {Name: "age", Orderable: true},
	}

	tests := []struct {
		name          string
		orderable     bool
		union         bool
		order         []Order
		defaultSort   map[string]string
		mockQuery     string
		expectedError error
	}{
		{
			name:      "with_orderable_disabled",
			orderable: false,
			mockQuery: "SELECT * FROM `users`",
		},
		{
			name:      "with_union_enabled",
			orderable: true,
			union:     true,
			mockQuery: "SELECT * FROM `users` ORDER BY `union_order`",
		},
		{
			name:      "with_user_defined_ordering",
			orderable: true,
			order:     []Order{{Column: 0, Dir: "ASC"}},
			mockQuery: "SELECT * FROM `users` ORDER BY `name`",
		},
		{
			name:        "with_default_sorting",
			orderable:   true,
			defaultSort: map[string]string{"age": "DESC"},
			mockQuery:   "SELECT * FROM `users` ORDER BY `age` DESC",
		},
		{
			name:      "invalid_column_len",
			orderable: true,
			order:     []Order{{Column: 1, Dir: "ASC"}},
			mockQuery: "SELECT * FROM `users`",
		},
		{
			name:      "whitelist_column",
			orderable: true,
			order:     []Order{{Column: 0, Dir: ""}, {Column: 1, Dir: ""}},
			mockQuery: "SELECT * FROM `users` ORDER BY `age`",
		},
		{
			name:        "empty_column_name_default_sort",
			orderable:   true,
			defaultSort: map[string]string{"name": "ASC"},
			mockQuery:   "SELECT * FROM `users` ORDER BY `name`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt.config.Orderable = tt.orderable
			dt.config.Union = tt.union
			dt.req.Order = tt.order
			dt.config.DefaultSort = tt.defaultSort
			dt.req.Columns = []ColumnRequest{
				{Data: "name"},
				{Data: "age"},
			}

			if tt.name == "invalid_column_len" {
				dt.req.Columns = []ColumnRequest{
					{Data: "name"},
				}
			}

			if tt.name == "whitelist_column" {
				dt.whitelistColumns = map[string]bool{"age": true}
			}

			if tt.name == "empty_column_name_default_sort" {
				dt.columnsMap = map[string]Column{
					"name": {Name: "", Data: "name", Orderable: true},
					"age":  {Name: "age", Data: "age", Orderable: true},
				}
			}

			mock.ExpectQuery(qm(tt.mockQuery)).
				WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).
					AddRow(1, "John Doe", 25))

			query := dt.tx.Model(&User{})
			query = dt.applyOrder(query)

			var users []User
			if err := query.Find(&users).Error; err != nil {
				t.Fatalf("failed to execute query: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestApplyPagination(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	dt := New(db)

	tests := []struct {
		name      string
		paginate  bool
		start     int
		length    int
		mockQuery string
		mockArgs  []driver.Value
	}{
		{
			name:      "without_pagination",
			paginate:  false,
			mockQuery: "SELECT * FROM `users`",
		},
		{
			name:      "with_pagination",
			paginate:  true,
			start:     10,
			length:    5,
			mockQuery: "SELECT * FROM `users` LIMIT ? OFFSET ?",
			mockArgs:  []driver.Value{5, 10},
		},
		{
			name:      "with_length_zero",
			paginate:  true,
			start:     10,
			length:    0,
			mockQuery: "SELECT * FROM `users` LIMIT ? OFFSET ?",
			mockArgs:  []driver.Value{0, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt.config.Paginate = tt.paginate
			dt.req.Start = tt.start
			dt.req.Length = tt.length

			mock.ExpectQuery(qm(tt.mockQuery)).
				WithArgs(tt.mockArgs...).
				WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).
					AddRow(1, "John Doe", 25))

			query := dt.tx.Model(&User{})
			query = dt.applyPagination(query)

			var users []User
			if err := query.Find(&users).Error; err != nil {
				t.Fatalf("failed to execute query: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestCheckComplexQuery(t *testing.T) {
	dbMock, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	tests := []struct {
		name     string
		query    string
		union    bool
		distinct bool
		groupBy  []string
		having   []string
	}{
		{
			name:  "without_complex_clauses",
			query: "SELECT * FROM `users`",
		},
		{
			name:  "with_union",
			query: "SELECT id, name, age FROM users WHERE age > 18 UNION SELECT id, name, age FROM users WHERE age <= 18",
			union: true,
		},
		{
			name:     "with_distinct",
			query:    "SELECT DISTINCT name FROM users",
			distinct: true,
		},
		{
			name:    "with_group_by",
			query:   "SELECT age FROM users GROUP BY age",
			groupBy: []string{"GROUP BY AGE"},
		},
		{
			name:    "with_group_by_and_having",
			query:   "SELECT age FROM users GROUP BY age HAVING COUNT(*) > 1",
			groupBy: []string{"GROUP BY AGE"},
			having:  []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := New(db)
			dt.tx = dt.tx.Raw(tt.query)

			dt.checkComplexQuery()

			if dt.config.Union != tt.union {
				t.Errorf("expected Union=%v, got %v", tt.union, dt.config.Union)
			}

			if dt.config.Distinct != tt.distinct {
				t.Errorf("expected Distinct=%v, got %v", tt.distinct, dt.config.Distinct)
			}

			if !reflect.DeepEqual(dt.config.GroupBy, tt.groupBy) {
				t.Errorf("expected GroupBy=%v, got %v", tt.groupBy, dt.config.GroupBy)
			}

			if !reflect.DeepEqual(dt.config.Having, tt.having) {
				t.Errorf("expected Having=%v, got %v", tt.having, dt.config.Having)
			}
		})
	}
}

func TestProcessQuery(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(mock sqlmock.Sqlmock)
		expectedError  bool
		expectedTotal  int64
		expectedFilter int64
	}{
		{
			name: "successful_query",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(qm("SELECT count(*) FROM `users`")).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

				mock.ExpectQuery(qm("SELECT count(*) FROM `users` WHERE (`id` LIKE ? OR `name` LIKE ? OR `age` LIKE ?)")).
					WithArgs("%John%", "%John%", "%John%").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

				mock.ExpectQuery(qm("SELECT * FROM `users` WHERE (`id` LIKE ? OR `name` LIKE ? OR `age` LIKE ?) LIMIT ?")).
					WithArgs("%John%", "%John%", "%John%", 10).
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).
						AddRow(1, "John Doe", 25).
						AddRow(2, "John Smith", 30))
			},
			expectedError:  false,
			expectedTotal:  25,
			expectedFilter: 10,
		},
		{
			name: "error_in_get_total_count",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(qm("SELECT count(*) FROM `users`")).
					WillReturnError(gorm.ErrInvalidData)
			},
			expectedError: true,
		},
		{
			name: "error_in_get_filtered_count",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(qm("SELECT count(*) FROM `users`")).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

				mock.ExpectQuery(qm("SELECT count(*) FROM `users` WHERE (`id` LIKE ? OR `name` LIKE ? OR `age` LIKE ?)")).
					WithArgs("%John%", "%John%", "%John%").
					WillReturnError(gorm.ErrInvalidData)
			},
			expectedError: true,
		},
		{
			name: "error_in_execute_query",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(qm("SELECT count(*) FROM `users`")).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

				mock.ExpectQuery(qm("SELECT count(*) FROM `users` WHERE (`id` LIKE ? OR `name` LIKE ? OR `age` LIKE ?)")).
					WithArgs("%John%", "%John%", "%John%").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

				mock.ExpectQuery(qm("SELECT * FROM `users` WHERE (`id` LIKE ? OR `name` LIKE ? OR `age` LIKE ?) LIMIT ?")).
					WithArgs("%John%", "%John%", "%John%", 10).
					WillReturnError(gorm.ErrInvalidData)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbMock, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer dbMock.Close()

			dialector := mysql.New(mysql.Config{
				Conn:                      dbMock,
				SkipInitializeWithVersion: true,
			})
			db, err := gorm.Open(dialector, &gorm.Config{})
			if err != nil {
				t.Fatalf("failed to open gorm DB: %v", err)
			}

			tt.mockSetup(mock)

			dt := New(db)
			dt.Req(Request{
				Draw: 1,
				Columns: []ColumnRequest{
					{Name: "id", Data: "id", Searchable: true},
					{Name: "name", Data: "name", Searchable: true},
					{Name: "age", Data: "age", Searchable: true},
				},
				Start:  0,
				Length: 10,
				Search: Search{
					Value: "John",
					Regex: false,
				},
			})

			dt.config.Orderable = true
			dt.config.Paginate = true
			dt.Model(User{})

			response, total, filtered, err := dt.processQuery()

			if tt.expectedError && err == nil {
				t.Fatal("expected an error but got nil")
			}

			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.expectedError {
				if total != tt.expectedTotal {
					t.Errorf("expected total=%d, got %d", tt.expectedTotal, total)
				}

				if filtered != tt.expectedFilter {
					t.Errorf("expected filtered=%d, got %d", tt.expectedFilter, filtered)
				}

				rawResponse, ok := response.([]map[string]any)
				if !ok {
					t.Fatalf("expected response to be of type []map[string]any, got %T", response)
				}

				expectedResponse := []map[string]any{
					{"id": 1, "name": "John Doe", "age": 25},
					{"id": 2, "name": "John Smith", "age": 30},
				}

				if !reflect.DeepEqual(normalizeResponse(rawResponse), normalizeResponse(expectedResponse)) {
					t.Errorf("expected response=%v, got %v", expectedResponse, rawResponse)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRaw(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbMock.Close()

	dialector := mysql.New(mysql.Config{
		Conn:                      dbMock,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	db = db.Table("users")

	mock.ExpectQuery(qm("SELECT count(*) FROM `users`")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

	mock.ExpectQuery(qm("SELECT count(*) FROM `users` WHERE (`ID` LIKE ? OR `Name` LIKE ? OR `Age` LIKE ? OR `Group` LIKE ?)")).
		WithArgs("%John%", "%John%", "%John%", "%John%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	mock.ExpectQuery(qm("SELECT * FROM `users` WHERE (`ID` LIKE ? OR `Name` LIKE ? OR `Age` LIKE ? OR `Group` LIKE ?) ORDER BY `ID` LIMIT ? OFFSET ?")).
		WithArgs("%John%", "%John%", "%John%", "%John%", 10, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age", "group"}).
			AddRow(1, "John Doe", 25, "A").
			AddRow(2, "John Smith", 30, "B"))

	dt := New(db)
	dt.Req(Request{
		Draw: 1,
		Columns: []ColumnRequest{
			{Name: "ID", Data: "id", Searchable: true, Orderable: true},
			{Name: "Name", Data: "name", Searchable: true, Orderable: true},
			{Name: "Age", Data: "age", Searchable: true, Orderable: true},
			{Name: "Group", Data: "group", Searchable: true, Orderable: true},
		},
		Start:  10,
		Length: 10,
		Search: Search{
			Value: "John",
			Regex: false,
		},
	})

	dt.config.Orderable = true
	dt.config.Paginate = true
	dt.config.DefaultSort = map[string]string{
		"id": "asc",
	}

	data, err := dt.Raw()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedResponse := []map[string]any{
		{"id": 1, "name": "John Doe", "age": 25, "group": "A"},
		{"id": 2, "name": "John Smith", "age": 30, "group": "B"},
	}

	normalizedResponse := normalizeResponse(data.([]map[string]any))
	normalizedExpectedResponse := normalizeResponse(expectedResponse)

	if !reflect.DeepEqual(normalizedResponse, normalizedExpectedResponse) {
		t.Errorf("expected response = %v, got %v", normalizedExpectedResponse, normalizedResponse)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestHasGroupByClause(t *testing.T) {
	tests := []struct {
		name  string
		query *gorm.DB
		want  bool
	}{
		{
			name: "with_group_by_clause",
			query: &gorm.DB{
				Statement: &gorm.Statement{
					Clauses: map[string]clause.Clause{
						queryGroupBy: {},
					},
				},
			},
			want: true,
		},
		{
			name: "without_group_by_clause",
			query: &gorm.DB{
				Statement: &gorm.Statement{
					Clauses: map[string]clause.Clause{},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasGroupByClause(tt.query); got != tt.want {
				t.Errorf("hasGroupByClause() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasHavingClause(t *testing.T) {
	tests := []struct {
		name  string
		query *gorm.DB
		want  bool
	}{
		{
			name: "with_having_clause",
			query: &gorm.DB{
				Statement: &gorm.Statement{
					Clauses: map[string]clause.Clause{
						queryHaving: {},
					},
				},
			},
			want: true,
		},
		{
			name: "without_having_clause",
			query: &gorm.DB{
				Statement: &gorm.Statement{
					Clauses: map[string]clause.Clause{},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasHavingClause(tt.query); got != tt.want {
				t.Errorf("hasHavingClause() = %v, want %v", got, tt.want)
			}
		})
	}
}
