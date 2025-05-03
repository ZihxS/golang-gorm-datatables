package datatables

import (
	"fmt"
	"net/http"
	"strconv"
)

// Search represents the search criteria for a DataTable.
//
// Fields:
//   - Value: The search term or value to be used.
//   - Regex: A boolean indicating whether the search should be treated as a regular expression.
type Search struct {
	Value string `form:"value"`
	Regex bool   `form:"regex"`
}

// Order specifies the ordering criteria for a DataTable column.
//
// Fields:
//   - Column: The index of the column to be ordered.
//   - Dir: The direction of ordering, either "asc" for ascending or "desc" for descending.
type Order struct {
	Column int    `form:"column"`
	Dir    string `form:"dir"`
}

// ColumnRequest represents a request for a DataTable column configuration.
//
// Fields:
//   - Searchable: Indicates if the column is searchable.
//   - Orderable: Indicates if the column can be ordered.
//   - Data: The data property name of the column.
//   - Name: The display name of the column.
//   - Search: The search criteria applied to the column.
type ColumnRequest struct {
	Searchable bool   `form:"searchable"`
	Orderable  bool   `form:"orderable"`
	Data       string `form:"data"`
	Name       string `form:"name"`
	Search     Search `form:"search"`
}

// Request represents a DataTables request.
//
// Fields:
//   - Draw: The draw counter for this request.
//   - Start: The start position for this request.
//   - Length: The length of the request.
//   - Search: The search criteria for this request.
//   - Order: The ordering criteria for this request.
//   - Columns: The columns to be processed for this request.
type Request struct {
	Draw    int             `form:"draw"`
	Start   int             `form:"start"`
	Length  int             `form:"length"`
	Search  Search          `form:"search"`
	Order   []Order         `form:"order"`
	Columns []ColumnRequest `form:"columns"`
}

// ParseRequest parses a DataTables request from the given http request.
//
// It will automatically parse the draw, start, length, search, order, and columns
// parameters from the request. The request is validated and an error is returned if
// any part of the request is invalid.
//
// The function returns the parsed request and nil if the request is valid,
// otherwise it returns nil and an error.
func ParseRequest(r *http.Request) (*Request, error) {
	var (
		err  error
		data Request
	)

	_ = r.ParseForm()

	data.Draw, err = strconv.Atoi(r.Form.Get("draw"))
	if err != nil {
		return nil, fmt.Errorf("invalid value for draw: %v", err)
	}
	data.Start, err = strconv.Atoi(r.Form.Get("start"))
	if err != nil {
		return nil, fmt.Errorf("invalid value for start: %v", err)
	}
	data.Length, _ = strconv.Atoi(r.Form.Get("length"))
	data.Search.Value = r.Form.Get("search[value]")
	data.Search.Regex, err = strconv.ParseBool(r.Form.Get("search[regex]"))
	if err != nil {
		return nil, fmt.Errorf("invalid value for search[regex]: %v", err)
	}

	columnCount := 0
	for {
		columnName := r.Form.Get(fmt.Sprintf("columns[%d][data]", columnCount))
		if columnName == "" {
			break
		}

		column := ColumnRequest{
			Data:       columnName,
			Name:       r.Form.Get(fmt.Sprintf("columns[%d][name]", columnCount)),
			Searchable: r.Form.Get(fmt.Sprintf("columns[%d][searchable]", columnCount)) == "true",
			Orderable:  r.Form.Get(fmt.Sprintf("columns[%d][orderable]", columnCount)) == "true",
			Search: Search{
				Value: r.Form.Get(fmt.Sprintf("columns[%d][search][value]", columnCount)),
				Regex: r.Form.Get(fmt.Sprintf("columns[%d][search][regex]", columnCount)) == "true",
			},
		}
		data.Columns = append(data.Columns, column)
		columnCount++
	}

	orderCount := 0
	for {
		columnIndex := r.Form.Get(fmt.Sprintf("order[%d][column]", orderCount))
		if columnIndex == "" {
			break
		}

		col, _ := strconv.Atoi(columnIndex)
		dir := r.Form.Get(fmt.Sprintf("order[%d][dir]", orderCount))

		if col >= 0 && col < len(data.Columns) && data.Columns[col].Orderable {
			order := Order{
				Column: col,
				Dir:    dir,
			}
			data.Order = append(data.Order, order)
		}
		orderCount++
	}

	if len(data.Order) == 0 {
		defaultSort := Order{
			Column: 0,
			Dir:    "asc",
		}
		if len(data.Columns) > 0 && data.Columns[0].Orderable {
			data.Order = append(data.Order, defaultSort)
		}
	}

	return &data, nil
}
