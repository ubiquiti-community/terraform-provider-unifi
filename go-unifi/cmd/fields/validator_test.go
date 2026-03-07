package main

import (
	"strings"
	"testing"
)

func TestBuildStringValidators(t *testing.T) {
	gen := NewSpecificationGenerator("test")

	tests := []struct {
		name       string
		validation string
		wantCount  int
		wantType   string
	}{
		{
			name:       "empty validation",
			validation: "",
			wantCount:  0,
		},
		{
			name:       "pipe-separated values",
			validation: "wan|wan2",
			wantCount:  1,
			wantType:   "OneOf",
		},
		{
			name:       "length between pattern",
			validation: ".{0,128}",
			wantCount:  1,
			wantType:   "LengthBetween",
		},
		{
			name:       "length at least pattern",
			validation: ".{1,}",
			wantCount:  1,
			wantType:   "LengthAtLeast",
		},
		{
			name:       "exact length pattern",
			validation: ".{32}",
			wantCount:  1,
			wantType:   "LengthBetween",
		},
		{
			name:       "hex pattern 32 chars",
			validation: "[0-9A-Fa-f]{32}",
			wantCount:  1,
			wantType:   "LengthBetween",
		},
		{
			name:       "hex pattern 512 chars",
			validation: "[0-9A-Fa-f]{512}",
			wantCount:  1,
			wantType:   "LengthBetween",
		},
		{
			name:       "color hex pattern",
			validation: "^#(?:[0-9a-fA-F]{3}){1,2}$",
			wantCount:  1,
			wantType:   "LengthBetween",
		},
		{
			name:       "regex pattern",
			validation: "^[a-z]+$",
			wantCount:  1,
			wantType:   "RegexMatches",
		},
		{
			name:       "complex enum",
			validation: "corporate|guest|wan|vlan-only",
			wantCount:  1,
			wantType:   "OneOf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validators := gen.buildStringValidators(tt.validation)

			if len(validators) != tt.wantCount {
				t.Errorf("got %d validators, want %d", len(validators), tt.wantCount)
				return
			}

			if tt.wantCount > 0 && tt.wantType != "" {
				if validators[0].Custom == nil {
					t.Error("expected custom validator")
					return
				}

				schemaDefContains := validators[0].Custom.SchemaDefinition
				if tt.wantType == "OneOf" && !strings.Contains(schemaDefContains, "stringvalidator.OneOf") {
					t.Errorf("expected OneOf validator, got %s", schemaDefContains)
				}
				if tt.wantType == "RegexMatches" && !strings.Contains(schemaDefContains, "stringvalidator.RegexMatches") {
					t.Errorf("expected RegexMatches validator, got %s", schemaDefContains)
				}
				if tt.wantType == "LengthBetween" && !strings.Contains(schemaDefContains, "stringvalidator.LengthBetween") {
					t.Errorf("expected LengthBetween validator, got %s", schemaDefContains)
				}
				if tt.wantType == "LengthAtLeast" && !strings.Contains(schemaDefContains, "stringvalidator.LengthAtLeast") {
					t.Errorf("expected LengthAtLeast validator, got %s", schemaDefContains)
				}
			}
		})
	}
}

func TestBuildInt64Validators(t *testing.T) {
	gen := NewSpecificationGenerator("test")

	tests := []struct {
		name       string
		validation string
		wantCount  int
	}{
		{
			name:       "empty validation",
			validation: "",
			wantCount:  0,
		},
		{
			name:       "pipe-separated integers",
			validation: "1|11|15|16|17",
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validators := gen.buildInt64Validators(tt.validation)

			if len(validators) != tt.wantCount {
				t.Errorf("got %d validators, want %d", len(validators), tt.wantCount)
			}

			if tt.wantCount > 0 {
				if validators[0].Custom == nil {
					t.Error("expected custom validator")
					return
				}
				if !strings.Contains(validators[0].Custom.SchemaDefinition, "int64validator.OneOf") {
					t.Errorf("expected OneOf validator, got %s", validators[0].Custom.SchemaDefinition)
				}
			}
		})
	}
}
