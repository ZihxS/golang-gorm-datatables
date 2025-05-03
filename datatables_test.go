package datatables

import (
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestNew(t *testing.T) {
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

	dt := New(db)
	if dt.tx == nil {
		t.Error("expected tx to be initialized, got nil")
	}
	if !dt.config.Searchable || !dt.config.Orderable || !dt.config.Paginate {
		t.Errorf("expected Searchable, Orderable, Paginate to be true, got %v", dt.config)
	}
	if len(dt.additionalData) != 0 {
		t.Errorf("expected additionalData to be empty, got %v", dt.additionalData)
	}
	if len(dt.whitelistColumns) != 0 {
		t.Errorf("expected whitelistColumns to be empty, got %v", dt.whitelistColumns)
	}
	if len(dt.blacklistColumns) != 0 {
		t.Errorf("expected blacklistColumns to be empty, got %v", dt.blacklistColumns)
	}
	if len(dt.columnsMap) != 0 {
		t.Errorf("expected columnsMap to be initialized")
	}
}

func TestMake(t *testing.T) {
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

	mock.ExpectQuery(qm("SELECT count(*) FROM `users`")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(100)))

	mock.ExpectQuery(qm("SELECT count(*) FROM `users` WHERE (`id` LIKE ? OR `name` LIKE ? OR `age` LIKE ?)")).
		WithArgs("%John%", "%John%", "%John%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(50)))

	mock.ExpectQuery(qm("SELECT * FROM `users` WHERE (`id` LIKE ? OR `name` LIKE ? OR `age` LIKE ?) LIMIT ? OFFSET ?")).
		WithArgs("%John%", "%John%", "%John%", 10, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).
			AddRow(1, "John Doe", 25).
			AddRow(2, "Jane Smith", 30))

	dt := New(db)
	dt.Model(&User{})
	dt.Req(Request{
		Draw:   1,
		Start:  10,
		Length: 10,
		Search: Search{Value: "John", Regex: false},
		Columns: []ColumnRequest{
			{Name: "id", Data: "id", Searchable: true, Orderable: true},
			{Name: "name", Data: "name", Searchable: true, Orderable: true},
			{Name: "age", Data: "age", Searchable: true, Orderable: true},
		},
	})

	dt.selectedColumns = []string{"no", "id", "name", "age", "group_name", "created_at"}

	t.Run("successful_process_query", func(t *testing.T) {
		column := dt.columnsMap["name"]
		column.RenderFunc = func(row map[string]any) any {
			return "Rendered_" + row["name"].(string)
		}
		dt.columnsMap["name"] = column

		dt.WithNumber()

		response, err := dt.Make()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedResponse := map[string]any{
			"draw":            int64(1),
			"recordsTotal":    int64(100),
			"recordsFiltered": int64(50),
			"data": []map[string]any{
				{"no": 11, "id": int64(1), "name": "Rendered_John Doe", "age": int64(25)},
				{"no": 12, "id": int64(2), "name": "Rendered_Jane Smith", "age": int64(30)},
			},
		}

		normalizedResponse := normalizeResponseMake(response)
		normalizedExpectedResponse := normalizeResponseMake(expectedResponse)

		if !reflect.DeepEqual(normalizedResponse, normalizedExpectedResponse) {
			t.Errorf("expected response = %v, got %v", normalizedExpectedResponse, normalizedResponse)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("process_query_error", func(t *testing.T) {
		mock.ExpectQuery(qm("SELECT count(*) FROM `users`")).
			WillReturnError(gorm.ErrInvalidData)

		_, err := dt.Make()
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err != gorm.ErrInvalidData {
			t.Errorf("expected error %v, got %v", gorm.ErrInvalidData, err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func TestMakeValidationError(t *testing.T) {
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

	dt := New(db)
	_, err = dt.Make()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if err.Error() != "model is required" {
		t.Errorf("expected error 'model is required', got '%v'", err)
	}
}
