package datatables

import (
	"maps"
	"runtime"
	"sync"

	"gorm.io/gorm"
)

// New returns a new DataTable with the given Gorm DB and default configuration.
func New(tx *gorm.DB) *DataTable {
	dt := &DataTable{
		tx: tx,
		config: Config{
			Searchable: true,
			Orderable:  true,
			Paginate:   true,
		},
		additionalData:   make(map[string]any),
		whitelistColumns: make(map[string]bool),
		blacklistColumns: make(map[string]bool),
		columnsMap:       make(map[string]Column),
	}
	dt.initColumnsMap()
	return dt
}

// Make processes the query and returns a DataTables compatible response.
//
// It will execute the following steps:
//  1. Validate the DataTable configuration.
//  2. Execute the query and get the total records count, filtered records count
//     and the actual data.
//  3. Run the custom column rendering functions in parallel.
//  4. Apply the row attributes in parallel.
//  5. Apply the custom columns in parallel.
//  6. If selected columns are defined, it will filter the columns for the response.
//  7. Merge the additional data into the response.
//  8. Return the response.
//
// The function returns a DataTables compatible response or an error if it
// occurs.
func (dt *DataTable) Make() (map[string]any, error) {
	if err := dt.Validate(); err != nil {
		return nil, err
	}

	data, total, filtered, err := dt.processQuery()
	if err != nil {
		return nil, err
	}

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		semChan   = make(chan struct{}, runtime.NumCPU()*2)
		dataSlice = data.([]map[string]any)
	)

	if noCol, ok := dt.columnsMap["no"]; ok {
		wg.Add(len(dataSlice))
		for i := range dataSlice {
			go func(i int) {
				defer wg.Done()
				semChan <- struct{}{}
				defer func() { <-semChan }()
				mu.Lock()
				defer mu.Unlock()
				row := dataSlice[i]
				row[noCol.Data] = dt.req.Start + i + 1
			}(i)
		}
	}

	wg.Add(len(dataSlice))
	for _, row := range dataSlice {
		go func(row map[string]any) {
			defer wg.Done()
			semChan <- struct{}{}
			defer func() { <-semChan }()
			mu.Lock()
			defer mu.Unlock()
			for _, col := range dt.columns {
				if renderFunc := dt.columnsMap[col.Data].RenderFunc; renderFunc != nil {
					row[col.Data] = renderFunc(row)
				}
			}
		}(row)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		dt.applyCustomColumns(dataSlice)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		dt.applyRowAttributes(dataSlice)
	}()

	wg.Wait()

	if len(dt.selectedColumns) > 0 {
		data = dt.FinalizeResponseColumns(dataSlice)
	}

	response := map[string]any{
		"draw":            dt.req.Draw,
		"recordsTotal":    total,
		"recordsFiltered": filtered,
		"data":            data,
	}
	maps.Copy(response, dt.additionalData)

	return response, nil
}
