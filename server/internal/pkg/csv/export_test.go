package csv

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type countingResponseWriter struct {
	gin.ResponseWriter
	writes int
}

func (writer *countingResponseWriter) Write(data []byte) (int, error) {
	writer.writes++
	return writer.ResponseWriter.Write(data)
}

func TestStreamCSVFlushesRowsInBatches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	countingWriter := &countingResponseWriter{ResponseWriter: ctx.Writer}
	ctx.Writer = countingWriter

	const rows = 64
	err := StreamCSV(ctx, []string{"id", "name"}, "items.csv", func(write RowWriter) error {
		for index := 0; index < rows; index++ {
			if err := write([]string{fmt.Sprint(index), fmt.Sprintf("item-%d", index)}); err != nil {
				return err
			}
		}
		return nil
	}, rows)
	if err != nil {
		t.Fatalf("StreamCSV returned error: %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "id,name\n0,item-0\n") || !strings.Contains(body, "63,item-63\n") {
		t.Fatalf("unexpected csv body: %q", body)
	}
	if countingWriter.writes > 4 {
		t.Fatalf("expected batched response writes, got %d writes for %d rows", countingWriter.writes, rows)
	}
}
