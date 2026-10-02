// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package retry

import "testing"

func TestNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name string
		e    *NotFoundError
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.want {
				t.Errorf("NotFoundError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFoundError_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		e       *NotFoundError
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.Unwrap(); (err != nil) != tt.wantErr {
				t.Errorf("NotFoundError.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnexpectedStateError_Error(t *testing.T) {
	tests := []struct {
		name string
		e    *UnexpectedStateError
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.want {
				t.Errorf("UnexpectedStateError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnexpectedStateError_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		e       *UnexpectedStateError
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.Unwrap(); (err != nil) != tt.wantErr {
				t.Errorf("UnexpectedStateError.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTimeoutError_Error(t *testing.T) {
	tests := []struct {
		name string
		e    *TimeoutError
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.want {
				t.Errorf("TimeoutError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeoutError_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		e       *TimeoutError
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.Unwrap(); (err != nil) != tt.wantErr {
				t.Errorf("TimeoutError.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
