package csv

import (
	"encoding/csv"
	"fmt"

	"github.com/gin-gonic/gin"
)

// UTF8 BOM for Excel compatibility
var UTF8BOM = []byte{0xEF, 0xBB, 0xBF}

// RowWriter writes one CSV row.
type RowWriter func([]string) error

// RowProducer streams CSV rows through the provided writer.
type RowProducer func(RowWriter) error

// Keep downloads streaming without forcing one response write per CSV row.
const streamCSVFlushEveryRows = 1024

// StreamCSV streams CSV data to HTTP response using standard library
// Uses chunked transfer encoding for streaming (no Content-Length)
func StreamCSV(c *gin.Context, headers []string, filename string, producer RowProducer, rowCount int64) error {
	_ = rowCount
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

	rowIndex := 0
	writeRow := func(fields []string) error {
		if err := writer.Write(fields); err != nil {
			return err
		}
		rowIndex++

		if rowIndex%streamCSVFlushEveryRows == 0 {
			writer.Flush()
			return writer.Error()
		}
		return nil
	}

	if err := producer(writeRow); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}
