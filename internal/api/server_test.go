package api

import (
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/workflow"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestCreateRouteRequiresKeyAndReplays(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	handler := New(workflow.New(store)).Handler()
	payload := []byte(`{"batch_id":"api-b1","site_code":"SITE","stratum_reference":"L2","collector_id":"collector","collected_at":"2026-01-02T03:04:05Z","baseline_oxygen_ppm":20,"baseline_temperature_c":8,"expected_revision":0}`)

	withoutKey := httptest.NewRequest(http.MethodPost, "/v1/batches", bytes.NewReader(payload))
	withoutKey.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, withoutKey)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("缺少幂等键时 status=%d body=%s", response.Code, response.Body.String())
	}

	firstBody := callAPI(t, handler, http.MethodPost, "/v1/batches", "api-create", payload, http.StatusCreated)
	replayBody := callAPI(t, handler, http.MethodPost, "/v1/batches", "api-create", payload, http.StatusCreated)
	if string(firstBody) != string(replayBody) {
		t.Fatalf("幂等响应不一致\nfirst=%s\nreplay=%s", firstBody, replayBody)
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/batches/api-b1", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("查询 status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestDecoderRejectsUnknownFields(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	handler := New(workflow.New(store)).Handler()
	payload := []byte(`{"site_code":"SITE","unknown":true}`)
	callAPI(t, handler, http.MethodPost, "/v1/batches", "bad-json", payload, http.StatusBadRequest)
}

func callAPI(t *testing.T, handler http.Handler, method, path, key string, payload []byte, want int) []byte {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	result, err := io.ReadAll(response.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != want {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, response.Code, want, result)
	}
	return result
}
