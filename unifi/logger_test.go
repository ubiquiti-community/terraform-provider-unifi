package unifi

import (
	"bytes"
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_DoesNotInheritRootFieldsByDefault(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	ctx = tflog.SetField(ctx, "root_only", "root-value")

	logger := NewLogger(ctx)
	logger.Info("inheritance check", "per_call", "value")

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if got := entry["@module"]; got != "provider.api-client" {
		t.Fatalf("expected api-client subsystem module, got %v", got)
	}
	if got := entry["per_call"]; got != "value" {
		t.Fatalf("expected per-call field to be preserved, got %v", got)
	}
	if _, ok := entry["root_only"]; ok {
		t.Fatalf("expected root field to be absent, got %v", entry["root_only"])
	}
}

func TestNewLogger_MasksSensitivePerCallFields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	logger.Debug(
		"masked fields",
		"unifi_api_key",
		"api-key-123",
		"unifi_password",
		"secret-password",
		"api_key",
		"common-api-key",
		"password",
		"common-password",
		"authorization",
		"Bearer abc123",
		"url",
		"https://unifi.example.com/api",
	)

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	for _, key := range []string{"unifi_api_key", "unifi_password", "api_key", "password", "authorization"} {
		if got := entry[key]; got != "***" {
			t.Fatalf("expected %s to be masked, got %v", key, got)
		}
	}
	if got := entry["url"]; got != "https://unifi.example.com/api" {
		t.Fatalf("expected non-sensitive field to remain unchanged, got %v", got)
	}
}

func TestUnifiLogger_Debug(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	logger.Debug("test debug message", "key", "value")

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one log entry")
	}
}

func TestUnifiLogger_Info(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	logger.Info("test info message", "key", "value")

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one log entry")
	}
}

func TestUnifiLogger_Warn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	logger.Warn("test warn message", "key", "value")

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one log entry")
	}
}

func TestUnifiLogger_Error(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	logger.Error("test error message", "key", "value")

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one log entry")
	}
}

func TestUnifiLogger_Printf(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	logger.Printf("formatted %s %d", "message", 42)

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one log entry")
	}
}

func TestUnifiLogger_OddKeysAndValues(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	// Odd number of keysAndValues should log an error instead of panicking.
	logger.Debug("odd keys", "key_without_value")

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected error log entry for odd keysAndValues")
	}
}

func TestUnifiLogger_NonStringKey(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	logger := NewLogger(ctx)

	// Non-string key should log an error instead of panicking.
	logger.Info("non-string key", 123, "value")

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected error log entry for non-string key")
	}
}

func TestUnifiLogger_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)

	logger := NewLogger(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			logger.Debug(
				"performing request",
				"method",
				"GET",
				"url",
				"https://unifi.example.com/api",
				"iter",
				iter,
				"unifi_password",
				"secret",
			)
		}(i)
	}
	wg.Wait()

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) != 100 {
		t.Fatalf("expected 100 log entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if got := entry["unifi_password"]; got != "***" {
			t.Fatalf("expected sensitive field to be masked, got %v", got)
		}
	}
}

func TestUnifiLogger_ConcurrentSharedFieldMasking(t *testing.T) {
	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	ctx = tflog.NewSubsystem(ctx, subsystem)
	ctx = tflog.SubsystemSetField(ctx, subsystem, "unifi_password", "secret")
	ctx = tflog.SubsystemMaskFieldValuesWithFieldKeys(ctx, subsystem, "unifi_password")

	logger := &UnifiLogger{ctx: ctx}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			logger.Debug("shared field masking", "iter", iter)
		}(i)
	}
	wg.Wait()

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) != 100 {
		t.Fatalf("expected 100 log entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if got := entry["unifi_password"]; got != "***" {
			t.Fatalf("expected shared field to be masked, got %v", got)
		}
	}
}

func TestUnifiLogger_ConcurrentMixedLevels(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)

	logger := NewLogger(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(5)
		go func(iter int) {
			defer wg.Done()
			logger.Debug("debug msg", "iter", iter, "unifi_password", "secret")
		}(i)
		go func(iter int) {
			defer wg.Done()
			logger.Info("info msg", "iter", iter, "unifi_password", "secret")
		}(i)
		go func(iter int) {
			defer wg.Done()
			logger.Warn("warn msg", "iter", iter, "unifi_password", "secret")
		}(i)
		go func(iter int) {
			defer wg.Done()
			logger.Error("error msg", "iter", iter, "unifi_password", "secret")
		}(i)
		go func(iter int) {
			defer wg.Done()
			logger.Printf("printf msg %d", iter)
		}(i)
	}
	wg.Wait()

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("failed to decode log entries: %v", err)
	}

	if len(entries) != 500 {
		t.Fatalf("expected 500 log entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if got := entry["unifi_password"]; got != "***" && got != nil {
			t.Fatalf("expected sensitive field to be masked or absent, got %v", got)
		}
	}
}

func TestUnifiLogger_log(t *testing.T) {
	type args struct {
		fn func()
	}
	tests := []struct {
		name string
		l    *UnifiLogger
		args args
	}{
		{
			name: "executes_fn",
			l: func() *UnifiLogger {
				var buf bytes.Buffer
				ctx := tflogtest.RootLogger(context.Background(), &buf)
				return NewLogger(ctx)
			}(),
			args: args{fn: func() {}},
		},
		{
			name: "fn_called_once",
			l: func() *UnifiLogger {
				var buf bytes.Buffer
				ctx := tflogtest.RootLogger(context.Background(), &buf)
				return NewLogger(ctx)
			}(),
			args: args{fn: func() {
				// side-effect-free no-op; just confirms no panic
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.log(tt.args.fn)
		})
	}
}

func Test_convertToFields(t *testing.T) {
	type args struct {
		keysAndValues []any
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr bool
	}{
		{
			name:    "empty",
			args:    args{keysAndValues: []any{}},
			want:    map[string]any{},
			wantErr: false,
		},
		{
			name:    "single_pair",
			args:    args{keysAndValues: []any{"key", "value"}},
			want:    map[string]any{"key": "value"},
			wantErr: false,
		},
		{
			name:    "multiple_pairs",
			args:    args{keysAndValues: []any{"a", 1, "b", true}},
			want:    map[string]any{"a": 1, "b": true},
			wantErr: false,
		},
		{
			name:    "odd_number_of_args",
			args:    args{keysAndValues: []any{"key_without_value"}},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "non_string_key",
			args:    args{keysAndValues: []any{42, "value"}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToFields(tt.args.keysAndValues)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertToFields() = %v, want %v", got, tt.want)
			}
		})
	}
}
