// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package retry

import (
	"context"
	"reflect"
	"testing"
)

func TestStateChangeConf_WaitForStateContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		conf    *StateChangeConf
		args    args
		want    any
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.conf.WaitForStateContext(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("StateChangeConf.WaitForStateContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StateChangeConf.WaitForStateContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateChangeConf_WaitForState(t *testing.T) {
	tests := []struct {
		name    string
		conf    *StateChangeConf
		want    any
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.conf.WaitForState()
			if (err != nil) != tt.wantErr {
				t.Errorf("StateChangeConf.WaitForState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StateChangeConf.WaitForState() = %v, want %v", got, tt.want)
			}
		})
	}
}
