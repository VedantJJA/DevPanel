package docker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleValidateBlueprint_InvalidInput(t *testing.T) {
	handler := HandleValidateBlueprint(nil)

	reqBody := []byte(`{"app_name":"","repo_url":""}`)
	req := httptest.NewRequest("POST", "/api/blueprints/validate", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected HTTP 400 Bad Request, got: %d", w.Code)
	}

	var res ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if res.Error == "" {
		t.Error("expected non-empty error message")
	}
}
