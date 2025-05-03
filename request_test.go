package datatables

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseRequest(t *testing.T) {
	type TestCaseParseRequest struct {
		Name           string
		Method         string
		QueryParams    url.Values
		RequestBody    string
		ExpectedError  bool
		ExpectedDraw   int
		ExpectedStart  int
		ExpectedLength int
		ExpectedSearch string
		ExpectedCols   []string
		ExpectedOrder  []Order
	}

	tests := []TestCaseParseRequest{
		{
			Name:           "valid_get_request",
			Method:         http.MethodGet,
			QueryParams:    url.Values{"draw": {"1"}, "start": {"0"}, "length": {"10"}, "search[value]": {"test"}, "columns[0][data]": {"no"}, "columns[0][name]": {"no"}, "columns[0][searchable]": {"true"}, "columns[0][orderable]": {"true"}, "columns[1][data]": {"name"}, "columns[1][name]": {"name"}, "columns[1][searchable]": {"true"}, "columns[1][orderable]": {"true"}, "order[0][column]": {"0"}, "order[0][dir]": {"asc"}, "search[regex]": {"false"}},
			ExpectedError:  false,
			ExpectedDraw:   1,
			ExpectedStart:  0,
			ExpectedLength: 10,
			ExpectedSearch: "test",
			ExpectedCols:   []string{"no", "name"},
			ExpectedOrder:  []Order{{Column: 0, Dir: "asc"}},
		},
		{
			Name:          "empty_get_query_params",
			Method:        http.MethodGet,
			QueryParams:   url.Values{},
			ExpectedError: true,
		},
		{
			Name:          "invalid_column_index_in_get",
			Method:        http.MethodGet,
			QueryParams:   url.Values{"draw": {"1"}, "start": {"0"}, "length": {"10"}, "order[0][column]": {"99"}, "order[0][dir]": {"asc"}},
			ExpectedError: true,
		},
		{
			Name:          "missing_required_fields_in_get",
			Method:        http.MethodGet,
			QueryParams:   url.Values{"draw": {"1"}},
			ExpectedError: true,
		},
		{
			Name:          "invalid_search_regex_in_get",
			Method:        http.MethodGet,
			QueryParams:   url.Values{"draw": {"1"}, "start": {"0"}, "length": {"10"}, "search[value]": {"test"}, "search[regex]": {"invalid"}},
			ExpectedError: true,
		},
		{
			Name:           "default_sorting_applied",
			Method:         http.MethodGet,
			QueryParams:    url.Values{"draw": {"1"}, "start": {"0"}, "length": {"10"}, "columns[0][data]": {"no"}, "columns[0][name]": {"no"}, "columns[0][searchable]": {"true"}, "columns[0][orderable]": {"true"}, "search[regex]": {"false"}},
			ExpectedError:  false,
			ExpectedDraw:   1,
			ExpectedStart:  0,
			ExpectedLength: 10,
			ExpectedSearch: "",
			ExpectedCols:   []string{"no"},
			ExpectedOrder:  []Order{{Column: 0, Dir: "asc"}},
		},
		{
			Name:           "valid_post_request",
			Method:         http.MethodPost,
			RequestBody:    "draw=1&start=0&length=10&search[value]=test&columns[0][data]=no&columns[0][name]=no&columns[0][searchable]=true&columns[0][orderable]=true&columns[1][data]=name&columns[1][name]=name&columns[1][searchable]=true&columns[1][orderable]=true&order[0][column]=0&order[0][dir]=asc&search[regex]=false",
			ExpectedError:  false,
			ExpectedDraw:   1,
			ExpectedStart:  0,
			ExpectedLength: 10,
			ExpectedSearch: "test",
			ExpectedCols:   []string{"no", "name"},
			ExpectedOrder:  []Order{{Column: 0, Dir: "asc"}},
		},
		{
			Name:          "empty_post_body",
			Method:        http.MethodPost,
			RequestBody:   "",
			ExpectedError: true,
		},
		{
			Name:          "missing_required_fields_in_post",
			Method:        http.MethodPost,
			RequestBody:   "draw=1",
			ExpectedError: true,
		},
		{
			Name:          "invalid_search_regex_in_post",
			Method:        http.MethodPost,
			RequestBody:   "draw=1&start=0&length=10&search[value]=test&search[regex]=invalid",
			ExpectedError: true,
		},
		{
			Name:           "default_sorting_applied_in_post",
			Method:         http.MethodPost,
			RequestBody:    "draw=1&start=0&length=10&columns[0][data]=no&columns[0][name]=no&columns[0][searchable]=true&columns[0][orderable]=true&search[regex]=true",
			ExpectedError:  false,
			ExpectedDraw:   1,
			ExpectedStart:  0,
			ExpectedLength: 10,
			ExpectedSearch: "",
			ExpectedCols:   []string{"no"},
			ExpectedOrder:  []Order{{Column: 0, Dir: "asc"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var req *http.Request

			if tt.Method == http.MethodPost {
				req = httptest.NewRequest(http.MethodPost, "/datatable", strings.NewReader(tt.RequestBody))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else if tt.Method == http.MethodGet {
				req = httptest.NewRequest(http.MethodGet, "/datatable?"+tt.QueryParams.Encode(), nil)
			} else {
				t.Fatalf("Unsupported HTTP method: %s", tt.Method)
			}

			parsedRequest, err := ParseRequest(req)

			if tt.ExpectedError && err == nil {
				t.Fatal("Expected an error but got nil")
			}

			if !tt.ExpectedError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.ExpectedError {
				return
			}

			if parsedRequest.Draw != tt.ExpectedDraw {
				t.Errorf("Expected Draw to be %d, got %d", tt.ExpectedDraw, parsedRequest.Draw)
			}

			if parsedRequest.Start != tt.ExpectedStart {
				t.Errorf("Expected Start to be %d, got %d", tt.ExpectedStart, parsedRequest.Start)
			}

			if parsedRequest.Length != tt.ExpectedLength {
				t.Errorf("Expected Length to be %d, got %d", tt.ExpectedLength, parsedRequest.Length)
			}

			if parsedRequest.Search.Value != tt.ExpectedSearch {
				t.Errorf("Expected Search.Value to be '%s', got '%s'", tt.ExpectedSearch, parsedRequest.Search.Value)
			}

			if len(parsedRequest.Columns) != len(tt.ExpectedCols) {
				t.Fatalf("Expected %d columns, got %d", len(tt.ExpectedCols), len(parsedRequest.Columns))
			}

			for i, col := range tt.ExpectedCols {
				if parsedRequest.Columns[i].Data != col {
					t.Errorf("Expected column %d Data to be '%s', got '%s'", i, col, parsedRequest.Columns[i].Data)
				}
			}

			if len(parsedRequest.Order) != len(tt.ExpectedOrder) {
				t.Fatalf("Expected %d order entries, got %d", len(tt.ExpectedOrder), len(parsedRequest.Order))
			}

			for i, order := range tt.ExpectedOrder {
				if parsedRequest.Order[i].Column != order.Column || parsedRequest.Order[i].Dir != order.Dir {
					t.Errorf("Expected order %d to be {Column: %d, Dir: '%s'}, got {Column: %d, Dir: '%s'}", i, order.Column, order.Dir, parsedRequest.Order[i].Column, parsedRequest.Order[i].Dir)
				}
			}
		})
	}
}
