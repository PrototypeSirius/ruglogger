package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLoggerChildFieldsAndJSONOutput(t *testing.T) {
	t.Cleanup(ResetDefaultForTest)

	var output bytes.Buffer
	log := MustNew(
		WithOutput(&output),
		WithFormat(FormatJSON),
		WithTimeFormat(time.RFC3339),
		WithLevel(LevelDebug),
		WithField("service", "billing"),
	)

	log.WithFields(Fields{
		"component": "worker",
	}).Info("processed payment", Fields{
		"attempt": 2,
	})

	record := decodeSingleRecord(t, output.String())

	if record["message"] != "processed payment" {
		t.Fatalf("unexpected message: %#v", record["message"])
	}
	if record["level"] != "INFO" {
		t.Fatalf("unexpected level: %#v", record["level"])
	}
	if record["service"] != "billing" {
		t.Fatalf("missing default field: %#v", record["service"])
	}
	if record["component"] != "worker" {
		t.Fatalf("missing child field: %#v", record["component"])
	}
	if record["attempt"] != float64(2) {
		t.Fatalf("missing request field: %#v", record["attempt"])
	}
}

func TestInitWithOptionsReplacesDefaultLogger(t *testing.T) {
	t.Cleanup(ResetDefaultForTest)

	var first bytes.Buffer
	if err := InitWithOptions(
		WithOutput(&first),
		WithFormat(FormatJSON),
	); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	Info("first", nil)

	var second bytes.Buffer
	if err := InitWithOptions(
		WithOutput(&second),
		WithFormat(FormatJSON),
	); err != nil {
		t.Fatalf("second init failed: %v", err)
	}

	Info("second", nil)

	if !strings.Contains(first.String(), `"message":"first"`) {
		t.Fatalf("expected first logger to contain first message, got %q", first.String())
	}
	if strings.Contains(first.String(), `"message":"second"`) {
		t.Fatalf("expected second message to be absent from first logger, got %q", first.String())
	}
	if !strings.Contains(second.String(), `"message":"second"`) {
		t.Fatalf("expected second logger to contain second message, got %q", second.String())
	}
}

func TestFatalUsesExitFuncWithoutTerminatingProcess(t *testing.T) {
	var output bytes.Buffer
	exitCode := 0

	log := MustNew(
		WithOutput(&output),
		WithFormat(FormatJSON),
		WithExitFunc(func(code int) {
			exitCode = code
		}),
	)

	log.Fatal("fatal failure", errors.New("boom"), 42, Fields{"job": "sync"})

	record := decodeSingleRecord(t, output.String())
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if record["message"] != "fatal failure" {
		t.Fatalf("unexpected message: %#v", record["message"])
	}
	if record["error"] != "boom" {
		t.Fatalf("unexpected error: %#v", record["error"])
	}
	if record["app_code"] != float64(42) {
		t.Fatalf("unexpected app code: %#v", record["app_code"])
	}
}

func decodeSingleRecord(t *testing.T, raw string) map[string]any {
	t.Helper()

	line := strings.TrimSpace(raw)
	if line == "" {
		t.Fatal("expected log output, got empty string")
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("failed to unmarshal log record %q: %v", line, err)
	}
	return record
}
