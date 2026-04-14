package apperror

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAppErrorMarshalJSONIncludesDetails(t *testing.T) {
	appErr := BadRequestError(errors.New("invalid id"), 1001, "Invalid user ID").WithDetails(map[string]any{
		"field": "id",
	})

	data, err := json.Marshal(appErr)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if payload["message"] != "Invalid user ID" {
		t.Fatalf("unexpected message: %#v", payload["message"])
	}
	if payload["app_code"] != float64(1001) {
		t.Fatalf("unexpected app code: %#v", payload["app_code"])
	}
	details, ok := payload["details"].(map[string]any)
	if !ok {
		t.Fatalf("expected details map, got %#v", payload["details"])
	}
	if details["field"] != "id" {
		t.Fatalf("unexpected details: %#v", details)
	}
}

func TestAsUnwrapsWrappedAppError(t *testing.T) {
	original := errors.New("db failure")
	appErr := SystemError(original, 9000, "")
	wrapped := errors.Join(errors.New("context"), appErr)

	resolved, ok := As(wrapped)
	if !ok {
		t.Fatal("expected As to resolve wrapped AppError")
	}
	if resolved.AppCode != 9000 {
		t.Fatalf("unexpected app code: %d", resolved.AppCode)
	}
	if !errors.Is(appErr, original) {
		t.Fatal("expected AppError to unwrap to original error")
	}
}
