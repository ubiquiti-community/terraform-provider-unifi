package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	subsystem = "api-client"
)

type UnifiLogger struct {
	baseCtx context.Context
}

func NewLogger(ctx context.Context) *UnifiLogger {
	return &UnifiLogger{
		baseCtx: ctx,
	}
}

func (l *UnifiLogger) ctx() context.Context {
	// Create a fresh subsystem context each time
	return tflog.NewSubsystem(l.baseCtx, subsystem)
}

func (l *UnifiLogger) Error(msg string, keysAndValues ...any) {
	ctx := l.ctx()

	fields, err := convertToFields(keysAndValues)
	if err != nil {
		tflog.SubsystemError(
			ctx,
			subsystem,
			fmt.Sprintf("invalid log key-value pairs: %s", err),
		)
	}
	tflog.SubsystemError(ctx, subsystem, msg, fields)
}

func (l *UnifiLogger) Printf(format string, v ...any) {
	tflog.SubsystemInfo(l.ctx(), subsystem, fmt.Sprintf(format, v...))
}

func (l *UnifiLogger) Info(msg string, keysAndValues ...any) {
	ctx := l.ctx()

	fields, err := convertToFields(keysAndValues)
	if err != nil {
		tflog.SubsystemError(
			ctx,
			subsystem,
			fmt.Sprintf("invalid log key-value pairs: %s", err),
		)
	}
	tflog.SubsystemInfo(ctx, subsystem, msg, fields)
}

func (l *UnifiLogger) Debug(msg string, keysAndValues ...any) {
	ctx := l.ctx()

	fields, err := convertToFields(keysAndValues)
	if err != nil {
		tflog.SubsystemError(
			ctx,
			subsystem,
			fmt.Sprintf("invalid log key-value pairs: %s", err),
		)
	}
	tflog.SubsystemDebug(ctx, subsystem, msg, fields)
}

func (l *UnifiLogger) Warn(msg string, keysAndValues ...any) {
	ctx := l.ctx()

	fields, err := convertToFields(keysAndValues)
	if err != nil {
		tflog.SubsystemError(
			ctx,
			subsystem,
			fmt.Sprintf("invalid log key-value pairs: %s", err),
		)
	}
	tflog.SubsystemWarn(ctx, subsystem, msg, fields)
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
