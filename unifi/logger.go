package unifi

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	subsystem = "api-client"
)

type UnifiLogger struct {
	ctx context.Context
	mu  sync.Mutex
}

func NewLogger(ctx context.Context) *UnifiLogger {
	return &UnifiLogger{
		ctx: tflog.NewSubsystem(ctx, subsystem),
	}
}

// Factor the lock boilerplate into one place.
func (l *UnifiLogger) log(fn func()) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fn()
}

func (l *UnifiLogger) Error(msg string, keysAndValues ...any) {
	l.log(func() {
		fields, err := convertToFields(keysAndValues)
		if err != nil {
			tflog.SubsystemError(
				l.ctx,
				subsystem,
				fmt.Sprintf("invalid log key-value pairs: %s", err),
			)
		}
		tflog.SubsystemError(l.ctx, subsystem, msg, fields)
	})
}

func (l *UnifiLogger) Printf(format string, v ...any) {
	l.log(func() {
		tflog.SubsystemInfo(l.ctx, subsystem, fmt.Sprintf(format, v...))
	})
}

func (l *UnifiLogger) Info(msg string, keysAndValues ...any) {
	l.log(func() {
		fields, err := convertToFields(keysAndValues)
		if err != nil {
			tflog.SubsystemError(
				l.ctx,
				subsystem,
				fmt.Sprintf("invalid log key-value pairs: %s", err),
			)
		}
		tflog.SubsystemInfo(l.ctx, subsystem, msg, fields)
	})
}

func (l *UnifiLogger) Debug(msg string, keysAndValues ...any) {
	l.log(func() {
		fields, err := convertToFields(keysAndValues)
		if err != nil {
			tflog.SubsystemError(
				l.ctx,
				subsystem,
				fmt.Sprintf("invalid log key-value pairs: %s", err),
			)
		}
		tflog.SubsystemDebug(l.ctx, subsystem, msg, fields)
	})
}

func (l *UnifiLogger) Warn(msg string, keysAndValues ...any) {
	l.log(func() {
		fields, err := convertToFields(keysAndValues)
		if err != nil {
			tflog.SubsystemError(
				l.ctx,
				subsystem,
				fmt.Sprintf("invalid log key-value pairs: %s", err),
			)
		}
		tflog.SubsystemWarn(l.ctx, subsystem, msg, fields)
	})
}

func convertToFields(keysAndValues []any) (map[string]any, error) {
	additionalFields := make(map[string]any, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 >= len(keysAndValues) {
			return nil, fmt.Errorf("missing value for key %s", keysAndValues[i])
		}

		if key, ok := keysAndValues[i].(string); ok {
			additionalFields[key] = keysAndValues[i+1]
		} else {
			return nil, fmt.Errorf("key %v is not a string", keysAndValues[i])
		}
	}
	return additionalFields, nil
}
