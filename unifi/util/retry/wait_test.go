// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package retry

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestRetryContext(t *testing.T) {
	type args struct {
		ctx     context.Context
		timeout time.Duration
		f       RetryFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RetryContext(tt.args.ctx, tt.args.timeout, tt.args.f); (err != nil) != tt.wantErr {
				t.Errorf("RetryContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetry(t *testing.T) {
	type args struct {
		timeout time.Duration
		f       RetryFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Retry(tt.args.timeout, tt.args.f); (err != nil) != tt.wantErr {
				t.Errorf("Retry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetryError_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		e       *RetryError
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.Unwrap(); (err != nil) != tt.wantErr {
				t.Errorf("RetryError.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetryableError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *RetryError
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RetryableError(tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNonRetryableError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *RetryError
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NonRetryableError(tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NonRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}
