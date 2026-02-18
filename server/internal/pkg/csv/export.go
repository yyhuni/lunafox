package csv

import (
	"database/sql"
	"encoding/csv"
	"fmt"

	"github.com/gin-gonic/gin"
)

// UTF8 BOM for Excel compatibility
var UTF8BOM = []byte{0xEF, 0xBB, 0xBF}

// RowMapper converts a database row to CSV fields
type RowMapper func(*sql.Rows) ([]string, error)

// StreamCSV streams CSV data to HTTP response using standard library
// Uses chunked transfer encoding for streaming (no Content-Length)
func StreamCSV(c *gin.Context, rows *sql.Rows, headers []string, filename string, mapper RowMapper, rowCount int64) error {
	defer func() { _ = rows.Close() }()

	// Set response headers for streaming download
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	// Don't set Content-Length - use chunked transfer encoding for streaming

	// Write UTF-8 BOM for Excel
	if _, err := c.Writer.Write(UTF8BOM); err != nil {
		return err
	}

	// Use standard library csv.Writer
	writer := csv.NewWriter(c.Writer)

	// Write CSV header
	if err := writer.Write(headers); err != nil {
		return err
	}
	writer.Flush()

	// Stream rows
	for rows.Next() {
		fields, err := mapper(rows)
		if err != nil {
			return err
		}

		if err := writer.Write(fields); err != nil {
			return err
		}

		// Flush periodically for large exports
		writer.Flush()
	}

	return rows.Err()
}
