package sql

import (
	"database/sql"
	"fmt"
	"strings"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
	_ "modernc.org/sqlite"
)

// Engine holds an in-memory SQLite database loaded from a DataSet.
type Engine struct {
	db        *sql.DB
	tableName string
}

// NewEngine creates a SQLite in-memory database and loads the DataSet into it.
func NewEngine(data *csvpkg.DataSet) (*Engine, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	tableName := "data"

	// Build CREATE TABLE with all columns as TEXT
	var cols []string
	for _, h := range data.Headers {
		cols = append(cols, fmt.Sprintf("[%s] TEXT", h))
	}
	createSQL := fmt.Sprintf("CREATE TABLE [%s] (%s)", tableName, strings.Join(cols, ", "))
	if _, err := db.Exec(createSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	// Insert rows in batches
	if len(data.Rows) > 0 {
		placeholders := "(" + strings.Repeat("?,", len(data.Headers)-1) + "?)"
		batchSize := 500
		for i := 0; i < len(data.Rows); i += batchSize {
			end := i + batchSize
			if end > len(data.Rows) {
				end = len(data.Rows)
			}
			batch := data.Rows[i:end]

			var allPlaceholders []string
			var args []interface{}
			for _, row := range batch {
				allPlaceholders = append(allPlaceholders, placeholders)
				for j := range data.Headers {
					if j < len(row) {
						args = append(args, row[j])
					} else {
						args = append(args, "")
					}
				}
			}

			insertSQL := fmt.Sprintf("INSERT INTO [%s] VALUES %s", tableName, strings.Join(allPlaceholders, ","))
			if _, err := db.Exec(insertSQL, args...); err != nil {
				db.Close()
				return nil, fmt.Errorf("insert batch: %w", err)
			}
		}
	}

	return &Engine{db: db, tableName: tableName}, nil
}

// Query executes a SQL query and returns the result as a DataSet.
func (e *Engine) Query(query string) (*csvpkg.DataSet, error) {
	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	ds := &csvpkg.DataSet{
		Headers:   columns,
		HasHeader: true,
		Delimiter: ',',
		FilePath:  "(query result)",
	}

	for rows.Next() {
		vals := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, len(columns))
		for i, v := range vals {
			if v == nil {
				row[i] = ""
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		ds.Rows = append(ds.Rows, row)
	}

	ds.OrigIndex = make([]int, len(ds.Rows))
	for i := range ds.Rows {
		ds.OrigIndex[i] = i + 1
	}

	return ds, rows.Err()
}

// TableName returns the name of the loaded table.
func (e *Engine) TableName() string {
	return e.tableName
}

// Close releases the database resources.
func (e *Engine) Close() {
	if e.db != nil {
		e.db.Close()
	}
}
