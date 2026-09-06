package dto

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindJSONRejectsInvalidUTF8BeforeJSONDecode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/", func(c *gin.Context) {
		var request struct {
			URL string `json:"url"`
		}
		if !BindJSON(c, &request) {
			return
		}
		c.Status(http.StatusNoContent)
	})

	payload := append([]byte(`{"url":"https://example.com/`), 0xff)
	payload = append(payload, []byte(`"}`)...)
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid UTF-8 request code = %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestBindJSONRejectsUnpairedSurrogateBeforeJSONDecode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/", func(c *gin.Context) {
		var request struct {
			URL string `json:"url"`
		}
		if !BindJSON(c, &request) {
			return
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"url":"https://example.com/\uD800"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unpaired-surrogate request code = %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestJSONTextValidatingReadCloserKeepsStateAcrossReadBoundaries(t *testing.T) {
	reader := &jsonTextValidatingReadCloser{ReadCloser: &oneByteReadCloser{Reader: bytes.NewReader([]byte(`{"url":"https://example.com/\uD83D\uDE00/界"}`))}}
	payload, err := io.ReadAll(reader)
	if err != nil || string(payload) != `{"url":"https://example.com/\uD83D\uDE00/界"}` {
		t.Fatalf("streaming JSON-text validation payload=%q err=%v", payload, err)
	}
}

type oneByteReadCloser struct {
	io.Reader
}

func (reader *oneByteReadCloser) Read(buffer []byte) (int, error) {
	if len(buffer) > 1 {
		buffer = buffer[:1]
	}
	return reader.Reader.Read(buffer)
}

func (reader *oneByteReadCloser) Close() error { return nil }
