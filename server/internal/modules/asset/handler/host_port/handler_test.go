package hostport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type hostPortStoreStub struct {
	rows         []assetdomain.IPAggregationRow
	total        int64
	count        int64
	scannedItem  *assetdomain.HostPort
	hostsByIP    map[string][]string
	portsByIP    map[string][]int
	listErr      error
	hostPortErr  error
	streamErr    error
	streamIPsErr error
	countErr     error
	scanErr      error
	deleteErr    error
}

func (stub *hostPortStoreStub) GetIPAggregation(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.IPAggregationRow, int64, error) {
	_ = targetID
	_ = page
	_ = pageSize
	_ = filter
	_ = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.IPAggregationRow(nil), stub.rows...), stub.total, nil
}

func (stub *hostPortStoreStub) GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error) {
	_ = targetID
	_ = filter
	if stub.hostPortErr != nil {
		return nil, nil, stub.hostPortErr
	}
	return append([]string(nil), stub.hostsByIP[ip]...), append([]int(nil), stub.portsByIP[ip]...), nil
}

func (stub *hostPortStoreStub) ListPortOptionsByTargetID(targetID int) ([]assetdomain.FilterOption, error) {
	_ = targetID
	return []assetdomain.FilterOption{{Value: "443", Label: "443", Count: 1}}, nil
}

func (stub *hostPortStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.HostPort) error) error {
	_ = ctx
	_ = targetID
	if stub.streamErr != nil {
		return stub.streamErr
	}
	if stub.scanErr != nil {
		return stub.scanErr
	}
	if stub.scannedItem != nil {
		copyItem := *stub.scannedItem
		return visit(copyItem)
	}
	return nil
}

func (stub *hostPortStoreStub) ForEachByTargetIDAndIPs(ctx context.Context, targetID int, ips []string, visit func(assetdomain.HostPort) error) error {
	_ = ctx
	_ = targetID
	_ = ips
	if stub.streamIPsErr != nil {
		return stub.streamIPsErr
	}
	if stub.scanErr != nil {
		return stub.scanErr
	}
	if stub.scannedItem != nil {
		copyItem := *stub.scannedItem
		return visit(copyItem)
	}
	return nil
}

func (stub *hostPortStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *hostPortStoreStub) BatchUpsert(items []assetdomain.HostPort) (int64, error) {
	return int64(len(items)), nil
}

func (stub *hostPortStoreStub) BatchUpsertContext(_ context.Context, items []assetdomain.HostPort) (int64, error) {
	return stub.BatchUpsert(items)
}

func (stub *hostPortStoreStub) DeleteByIPs(ips []string) (int64, error) {
	if stub.deleteErr != nil {
		return 0, stub.deleteErr
	}
	return int64(len(ips)), nil
}

type hostPortLookupStub struct {
	target *assetdomain.TargetRef
	err    error
}

func (stub *hostPortLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	if stub.target == nil || stub.target.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *stub.target
	return &copyTarget, nil
}

func (stub *hostPortLookupStub) GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetActiveByID(id)
}

func TestHostPortHandlerCoverage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	store := &hostPortStoreStub{
		rows:        []assetdomain.IPAggregationRow{{IP: "1.1.1.1", CreatedAt: now}},
		total:       1,
		count:       1,
		scannedItem: &assetdomain.HostPort{ID: 1, TargetID: 1, IP: "1.1.1.1", Host: "a.example.com", Port: 443, CreatedAt: now},
		hostsByIP:   map[string][]string{"1.1.1.1": {"a.example.com"}},
		portsByIP:   map[string][]int{"1.1.1.1": {443}},
	}
	lookup := &hostPortLookupStub{target: &assetdomain.TargetRef{ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}
	handler := NewHostPortHandler(service.NewHostPortFacade(service.NewHostPortQueryService(store, lookup), service.NewHostPortCommandService(store, lookup)))

	if handler == nil || handler.svc == nil {
		t.Fatal("expected host-port handler")
	}

	t.Run("list invalid target", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.List, http.MethodGet, "/v1/targets/bad/hostPorts", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list invalid query", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.List, http.MethodGet, "/v1/targets/1/hostPorts?pageSize=abc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performHostPortRequest(t, handler.List, http.MethodGet, "/v1/targets/1/hostPorts", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("list success", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.List, http.MethodGet, "/v1/targets/1/hostPorts", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "1.1.1.1") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("filter options success", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.FilterOptions, http.MethodGet, "/v1/targets/1/hostPorts/filterOptions?field=port", gin.Params{{Key: "target", Value: "1"}}, "", "")
		body := recorder.Body.String()
		if recorder.Code != http.StatusOK || !strings.Contains(body, `"results"`) || !strings.Contains(body, `"value":"443"`) || !strings.Contains(body, `"count":1`) {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, body)
		}
	})

	t.Run("filter options rejects unsupported field", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.FilterOptions, http.MethodGet, "/v1/targets/1/hostPorts/filterOptions?field=ip", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list nil hosts and ports become empty arrays", func(t *testing.T) {
		store.hostsByIP = map[string][]string{"1.1.1.1": nil}
		store.portsByIP = map[string][]int{"1.1.1.1": nil}
		recorder := performHostPortRequest(t, handler.List, http.MethodGet, "/v1/targets/1/hostPorts", gin.Params{{Key: "target", Value: "1"}}, "", "")
		body := recorder.Body.String()
		if recorder.Code != http.StatusOK || !strings.Contains(body, "\"hosts\":[]") || !strings.Contains(body, "\"ports\":[]") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, body)
		}
		store.hostsByIP = map[string][]string{"1.1.1.1": {"a.example.com"}}
		store.portsByIP = map[string][]int{"1.1.1.1": {443}}
	})

	t.Run("list internal error", func(t *testing.T) {
		store.listErr = fmt.Errorf("list failed")
		recorder := performHostPortRequest(t, handler.List, http.MethodGet, "/v1/targets/1/hostPorts", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.listErr = nil
	})

	t.Run("batch delete bind failure", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.BatchDelete, http.MethodPost, "/v1/hostPorts:batchDelete", nil, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch delete success", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.BatchDelete, http.MethodPost, "/v1/hostPorts:batchDelete", nil, `{"ips":["1.1.1.1"]}`, "application/json")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
	})

	t.Run("batch delete internal error", func(t *testing.T) {
		store.deleteErr = fmt.Errorf("delete failed")
		recorder := performHostPortRequest(t, handler.BatchDelete, http.MethodPost, "/v1/hostPorts:batchDelete", nil, `{"ips":["1.1.1.1"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.deleteErr = nil
	})

	t.Run("export invalid id", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.Export, http.MethodGet, "/v1/targets/bad/hostPorts/exportFiles/current", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("export target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performHostPortRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/hostPorts/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("export success with count", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/hostPorts/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "ip,host,port,created_at") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("export success with ips", func(t *testing.T) {
		recorder := performHostPortRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/hostPorts/exportFiles/current?ips=1.1.1.1", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
	})

	t.Run("export target not found with ips", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performHostPortRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/hostPorts/exportFiles/current?ips=1.1.1.1", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("export stream error", func(t *testing.T) {
		store.streamErr = fmt.Errorf("stream failed")
		recorder := performHostPortRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/hostPorts/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "ip,host,port,created_at") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		store.streamErr = nil
	})

	t.Run("export count error", func(t *testing.T) {
		store.countErr = fmt.Errorf("count failed")
		recorder := performHostPortRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/hostPorts/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.countErr = nil
	})

	t.Run("export scan error", func(t *testing.T) {
		store.scanErr = fmt.Errorf("scan failed")
		recorder := performHostPortRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/hostPorts/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "ip,host,port,created_at") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		store.scanErr = nil
	})
}

func performHostPortRequest(t *testing.T, handlerFunc gin.HandlerFunc, method, target string, params gin.Params, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	ctx.Request = req
	ctx.Params = params
	handlerFunc(ctx)
	ctx.Writer.WriteHeaderNow()
	return recorder
}
