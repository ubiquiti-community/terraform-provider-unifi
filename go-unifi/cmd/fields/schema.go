package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-codegen-spec/code"
	"github.com/hashicorp/terraform-plugin-codegen-spec/datasource"
	"github.com/hashicorp/terraform-plugin-codegen-spec/provider"
	"github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	"github.com/hashicorp/terraform-plugin-codegen-spec/schema"
	"github.com/hashicorp/terraform-plugin-codegen-spec/spec"
	"github.com/iancoleman/strcase"
	"github.com/ubiquiti-community/go-unifi/internal/fields"
)

const (
	SpecVersion       = "0.1"
	GoUnifiImportPath = "github.com/ubiquiti-community/go-unifi/unifi"
)

// SpecificationGenerator generates a Terraform provider specification from resources.
type SpecificationGenerator struct {
	ProviderName string
	Resources    []*ResourceInfo
}

// NewSpecificationGenerator creates a new specification generator.
func NewSpecificationGenerator(providerName string) *SpecificationGenerator {
	return &SpecificationGenerator{
		ProviderName: providerName,
		Resources:    make([]*ResourceInfo, 0),
	}
}

// AddResource adds a resource to the specification generator.
func (g *SpecificationGenerator) AddResource(r *ResourceInfo) {
	g.Resources = append(g.Resources, r)
}

// Generate creates the Terraform provider specification.
func (g *SpecificationGenerator) Generate() *spec.Specification {
	spec := &spec.Specification{
		Version: SpecVersion,
		Provider: &provider.Provider{
			Name: g.ProviderName,
			Schema: &provider.Schema{
				Attributes: g.generateProviderAttributes(),
			},
		},
		DataSources: make([]datasource.DataSource, 0),
		Resources:   make([]resource.Resource, 0),
	}

	// Sort resources by name for consistent output
	sortedResources := make([]*ResourceInfo, len(g.Resources))
	copy(sortedResources, g.Resources)
	sort.Slice(sortedResources, func(i, j int) bool {
		return sortedResources[i].StructName < sortedResources[j].StructName
	})

	for _, r := range sortedResources {
		// Skip settings for now - they have a different pattern
		if r.IsSetting() {
			continue
		}

		// Generate data source
		ds := g.generateDataSource(r)
		spec.DataSources = append(spec.DataSources, ds)

		// Generate resource
		res := g.generateResource(r)
		spec.Resources = append(spec.Resources, res)
	}

	return spec
}

// generateProviderAttributes creates the provider configuration attributes.
func (g *SpecificationGenerator) generateProviderAttributes() []provider.Attribute {
	return []provider.Attribute{
		{
			Name: "username",
			String: &provider.StringAttribute{
				OptionalRequired: "optional",
				Description:      ptr("Username for UniFi controller authentication"),
			},
		},
		{
			Name: "password",
			String: &provider.StringAttribute{
				OptionalRequired: "optional",
				Sensitive:        ptr(true),
				Description:      ptr("Password for UniFi controller authentication"),
			},
		},
		{
			Name: "api_url",
			String: &provider.StringAttribute{
				OptionalRequired: "optional",
				Description:      ptr("URL of the UniFi controller API"),
			},
		},
		{
			Name: "api_key",
			String: &provider.StringAttribute{
				OptionalRequired: "optional",
				Description:      ptr("API key for the Unifi controller. Can be specified with the `UNIFI_API_KEY` environment variable"),
				Sensitive:        ptr(true),
			},
		},
		{
			Name: "site",
			String: &provider.StringAttribute{
				OptionalRequired: "optional",
				Description:      ptr("Site name for the UniFi controller"),
			},
		},
		{
			Name: "allow_insecure",
			Bool: &provider.BoolAttribute{
				OptionalRequired: "optional",
				Description:      ptr("Allow insecure HTTPS connections to the UniFi controller"),
			},
		},
	}
}

// generateDataSource generates a data source specification from a resource.
func (g *SpecificationGenerator) generateDataSource(r *ResourceInfo) datasource.DataSource {
	name := toTerraformName(r.StructName)

	ds := datasource.DataSource{
		Name: name,
		Schema: &datasource.Schema{
			Attributes: g.generateDataSourceAttributes(r),
		},
	}

	return ds
}

// generateDataSourceAttributes generates data source attributes from a resource.
func (g *SpecificationGenerator) generateDataSourceAttributes(r *ResourceInfo) []datasource.Attribute {
	baseType := r.Types[r.StructName]
	if baseType == nil || baseType.Fields == nil {
		return nil
	}

	attrs := make([]datasource.Attribute, 0)

	// Sort fields by name for consistent output
	fieldNames := make([]string, 0, len(baseType.Fields))
	for name := range baseType.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		field := baseType.Fields[fieldName]
		if field == nil || strings.HasPrefix(fieldName, " ") || strings.HasSuffix(fieldName, "_Spacer") {
			continue
		}

		attr := g.fieldToDataSourceAttribute(r, field)
		if attr != nil {
			attrs = append(attrs, *attr)
		}
	}

	return attrs
}

// fieldToDataSourceAttribute converts a FieldInfo to a datasource.Attribute.
func (g *SpecificationGenerator) fieldToDataSourceAttribute(r *ResourceInfo, field *FieldInfo) *datasource.Attribute {
	if field == nil {
		return nil
	}

	// Use JSONName directly as it's already in the correct Terraform format
	name := toTerraformName(field.FieldName)
	_ = g.buildAssociatedExternalType(r, field)
	var externalType *schema.AssociatedExternalType = nil

	attr := &datasource.Attribute{
		Name: name,
	}

	// Handle array types
	if field.IsArray {
		if field.Fields != nil {
			// Nested object array - use list_nested
			nestedAttrs := g.generateNestedDataSourceAttributes(r, field)
			attr.ListNested = &datasource.ListNestedAttribute{
				ComputedOptionalRequired: "computed",
				NestedObject: datasource.NestedAttributeObject{
					AssociatedExternalType: externalType,
					Attributes:             nestedAttrs,
				},
			}
		} else {
			// Simple array - use list
			attr.List = &datasource.ListAttribute{
				ComputedOptionalRequired: "computed",
				ElementType:              g.fieldTypeToElementType(field.FieldType),
				AssociatedExternalType:   externalType,
			}
		}
		return attr
	}

	// Handle nested object types
	if field.Fields != nil {
		nestedAttrs := g.generateNestedDataSourceAttributes(r, field)
		attr.SingleNested = &datasource.SingleNestedAttribute{
			ComputedOptionalRequired: "computed",
			Attributes:               nestedAttrs,
			AssociatedExternalType:   externalType,
		}
		return attr
	}

	// Handle primitive types
	switch field.FieldType {
	case "bool":
		attr.Bool = &datasource.BoolAttribute{
			ComputedOptionalRequired: "computed",
			AssociatedExternalType:   externalType,
		}
	case "int64":
		intAttr := &datasource.Int64Attribute{
			ComputedOptionalRequired: "computed",
			AssociatedExternalType:   externalType,
		}
		// if validators := g.buildInt64Validators(field.FieldValidation); len(validators) > 0 {
		// 	intAttr.Validators = validators
		// }
		attr.Int64 = intAttr
	case "float64":
		attr.Float64 = &datasource.Float64Attribute{
			ComputedOptionalRequired: "computed",
			AssociatedExternalType:   externalType,
		}
	case "string":
		strAttr := &datasource.StringAttribute{
			ComputedOptionalRequired: "computed",
			AssociatedExternalType:   externalType,
		}
		// if validators := g.buildStringValidators(field.FieldValidation); len(validators) > 0 {
		// 	strAttr.Validators = validators
		// }
		attr.String = strAttr
	default:
		// Check if it's a custom type defined in Types
		if _, ok := r.Types[field.FieldType]; ok {
			nestedAttrs := g.generateNestedDataSourceAttributesFromType(r, field.FieldType)
			attr.SingleNested = &datasource.SingleNestedAttribute{
				ComputedOptionalRequired: "computed",
				Attributes:               nestedAttrs,
				AssociatedExternalType:   externalType,
			}
		} else {
			// Default to string for unknown types
			attr.String = &datasource.StringAttribute{
				ComputedOptionalRequired: "computed",
				AssociatedExternalType:   externalType,
			}
		}
	}

	return attr
}

// generateNestedDataSourceAttributes generates nested attributes for data sources.
func (g *SpecificationGenerator) generateNestedDataSourceAttributes(r *ResourceInfo, field *FieldInfo) []datasource.Attribute {
	if field.Fields == nil {
		return nil
	}

	attrs := make([]datasource.Attribute, 0)
	fieldNames := make([]string, 0, len(field.Fields))
	for name := range field.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		childField := field.Fields[fieldName]
		if childField == nil {
			continue
		}

		attr := g.fieldToDataSourceAttribute(r, childField)
		if attr != nil {
			attrs = append(attrs, *attr)
		}
	}

	return attrs
}

// generateNestedDataSourceAttributesFromType generates nested attributes from a type name.
func (g *SpecificationGenerator) generateNestedDataSourceAttributesFromType(r *ResourceInfo, typeName string) []datasource.Attribute {
	typeInfo, ok := r.Types[typeName]
	if !ok || typeInfo.Fields == nil {
		return nil
	}

	return g.generateNestedDataSourceAttributes(r, typeInfo)
}

// generateResource generates a resource specification from a Resource.
func (g *SpecificationGenerator) generateResource(r *ResourceInfo) resource.Resource {
	name := toTerraformName(r.StructName)

	res := resource.Resource{
		Name: name,
		Schema: &resource.Schema{
			Attributes: g.generateResourceAttributes(r),
		},
	}

	return res
}

// generateResourceAttributes generates resource attributes from a Resource.
func (g *SpecificationGenerator) generateResourceAttributes(r *ResourceInfo) []resource.Attribute {
	baseType := r.Types[r.StructName]
	if baseType == nil || baseType.Fields == nil {
		return nil
	}

	attrs := make([]resource.Attribute, 0)

	// Sort fields by name for consistent output
	fieldNames := make([]string, 0, len(baseType.Fields))
	for name := range baseType.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		field := baseType.Fields[fieldName]
		if field == nil || strings.HasPrefix(fieldName, " ") || strings.HasSuffix(fieldName, "_Spacer") {
			continue
		}

		attr := g.fieldToResourceAttribute(r, field)
		if attr != nil {
			attrs = append(attrs, *attr)
		}
	}

	return attrs
}

// fieldToResourceAttribute converts a FieldInfo to a ResourceAttribute.
func (g *SpecificationGenerator) fieldToResourceAttribute(r *ResourceInfo, field *FieldInfo) *resource.Attribute {
	if field == nil {
		return nil
	}

	// Use JSONName directly as it's already in the correct Terraform format
	name := toTerraformName(field.FieldName)
	_ = g.buildAssociatedExternalType(r, field)
	var externalType *schema.AssociatedExternalType = nil
	computedOptionalRequired := g.determineComputedOptionalRequired(field)

	attr := &resource.Attribute{
		Name: name,
	}

	// Handle array types
	if field.IsArray {
		if field.Fields != nil {
			// Nested object array - use list_nested
			nestedAttrs := g.generateNestedResourceAttributes(r, field)
			attr.ListNested = &resource.ListNestedAttribute{
				ComputedOptionalRequired: computedOptionalRequired,
				NestedObject: resource.NestedAttributeObject{
					AssociatedExternalType: externalType,
					Attributes:             nestedAttrs,
				},
			}
		} else {
			// Simple array - use list
			attr.List = &resource.ListAttribute{
				ComputedOptionalRequired: computedOptionalRequired,
				ElementType:              g.fieldTypeToElementType(field.FieldType),
				AssociatedExternalType:   externalType,
			}
		}
		return attr
	}

	// Handle nested object types
	if field.Fields != nil {
		nestedAttrs := g.generateNestedResourceAttributes(r, field)
		attr.SingleNested = &resource.SingleNestedAttribute{
			ComputedOptionalRequired: computedOptionalRequired,
			Attributes:               nestedAttrs,
			AssociatedExternalType:   externalType,
		}
		return attr
	}

	// Handle primitive types
	switch field.FieldType {
	case "bool":
		attr.Bool = &resource.BoolAttribute{
			ComputedOptionalRequired: computedOptionalRequired,
			AssociatedExternalType:   externalType,
		}
	case fields.Int:
		intAttr := &resource.Int64Attribute{
			ComputedOptionalRequired: computedOptionalRequired,
			AssociatedExternalType:   externalType,
		}
		// if validators := g.buildInt64Validators(field.FieldValidation); len(validators) > 0 {
		// 	intAttr.Validators = validators
		// }
		attr.Int64 = intAttr
	case "float64":
		attr.Float64 = &resource.Float64Attribute{
			ComputedOptionalRequired: computedOptionalRequired,
			AssociatedExternalType:   externalType,
		}
	case "string":
		strAttr := &resource.StringAttribute{
			ComputedOptionalRequired: computedOptionalRequired,
			AssociatedExternalType:   externalType,
		}
		// if validators := g.buildStringValidators(field.FieldValidation); len(validators) > 0 {
		// 	strAttr.Validators = validators
		// }
		attr.String = strAttr
	default:
		// Check if it's a custom type defined in Types
		if _, ok := r.Types[field.FieldType]; ok {
			nestedAttrs := g.generateNestedResourceAttributesFromType(r, field.FieldType)
			attr.SingleNested = &resource.SingleNestedAttribute{
				ComputedOptionalRequired: computedOptionalRequired,
				Attributes:               nestedAttrs,
				AssociatedExternalType:   externalType,
			}
		} else {
			// Default to string for unknown types
			attr.String = &resource.StringAttribute{
				ComputedOptionalRequired: computedOptionalRequired,
				AssociatedExternalType:   externalType,
			}
		}
	}

	return attr
}

// generateNestedResourceAttributes generates nested attributes for resources.
func (g *SpecificationGenerator) generateNestedResourceAttributes(r *ResourceInfo, field *FieldInfo) []resource.Attribute {
	if field.Fields == nil {
		return nil
	}

	attrs := make([]resource.Attribute, 0)
	fieldNames := make([]string, 0, len(field.Fields))
	for name := range field.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		childField := field.Fields[fieldName]
		if childField == nil {
			continue
		}

		attr := g.fieldToResourceAttribute(r, childField)
		if attr != nil {
			attrs = append(attrs, *attr)
		}
	}

	return attrs
}

// generateNestedResourceAttributesFromType generates nested attributes from a type name.
func (g *SpecificationGenerator) generateNestedResourceAttributesFromType(r *ResourceInfo, typeName string) []resource.Attribute {
	typeInfo, ok := r.Types[typeName]
	if !ok || typeInfo.Fields == nil {
		return nil
	}

	return g.generateNestedResourceAttributes(r, typeInfo)
}

// buildAssociatedExternalType creates an AssociatedExternalType for a field.
func (g *SpecificationGenerator) buildAssociatedExternalType(_ *ResourceInfo, field *FieldInfo) *schema.AssociatedExternalType {
	if field == nil {
		return nil
	}

	var typeName string

	// Build the full type name including pointer and array notation
	if field.IsArray {
		if field.Fields != nil {
			// Nested object array type
			typeName = field.FieldType
		} else {
			typeName = field.FieldType
		}
	} else if field.Fields != nil {
		// Nested object type
		if field.OmitEmpty && field.IsPointer {
			typeName = fmt.Sprintf("*%s", field.FieldType)
		} else {
			typeName = field.FieldType
		}
	} else {
		// Primitive type
		if field.OmitEmpty && field.IsPointer {
			typeName = fmt.Sprintf("*%s", field.FieldType)
		} else {
			typeName = field.FieldType
		}
	}

	if regexp.MustCompile(`string|bool|int64|float64`).MatchString(typeName) {
		return nil
	}

	return &schema.AssociatedExternalType{
		Import: &code.Import{
			Path: GoUnifiImportPath,
		},
		Type: typeName,
	}
}

// determineComputedOptionalRequired determines the computed_optional_required value for a field.
func (g *SpecificationGenerator) determineComputedOptionalRequired(field *FieldInfo) schema.ComputedOptionalRequired {
	// ID and SiteID are computed
	if field.FieldName == "ID" || field.FieldName == "SiteID" {
		return schema.Computed
	}

	// Hidden attributes are computed
	if field.FieldName == "Hidden" || field.FieldName == "HiddenID" ||
		field.FieldName == "NoDelete" || field.FieldName == "NoEdit" {
		return schema.Computed
	}

	// If OmitEmpty is true, the field is optional
	if field.OmitEmpty {
		return schema.ComputedOptional
	}

	return schema.Optional
}

// buildValidators creates validators from a FieldValidation string.
func (g *SpecificationGenerator) buildValidators(fieldType string, fieldValidation string) any { //nolint:golint,unused
	if fieldValidation == "" {
		return nil
	}

	// Parse validation string
	switch fieldType {
	case "string":
		return g.buildStringValidators(fieldValidation)
	case fields.Int:
		return g.buildInt64Validators(fieldValidation)
	case "float64":
		return g.buildFloat64Validators(fieldValidation)
	case "bool":
		// Bool validators are less common
		return nil
	default:
		return nil
	}
}

// buildStringValidators creates string validators from validation string.
func (g *SpecificationGenerator) buildStringValidators(validation string) []schema.StringValidator {
	if validation == "" {
		return nil
	}

	validators := make([]schema.StringValidator, 0)

	// Check if it's a pipe-separated list of values (enum)
	if strings.Contains(validation, "|") && !strings.HasPrefix(validation, "^") {
		// Extract values between pipes, handle regex-like patterns
		// Simple pattern: value1|value2|value3
		values := strings.Split(validation, "|")
		if len(values) > 1 {
			// Build OneOf validator
			quotedValues := make([]string, len(values))
			for i, v := range values {
				v = strings.TrimSpace(v)
				// Remove regex anchors and special chars if present
				v = strings.TrimPrefix(v, "^")
				v = strings.TrimSuffix(v, "$")
				v = strings.Trim(v, "()")
				quotedValues[i] = fmt.Sprintf(`"%s"`, v)
			}
			schemaDefinition := fmt.Sprintf("stringvalidator.OneOf(%s)", strings.Join(quotedValues, ", "))
			validators = append(validators, schema.StringValidator{
				Custom: &schema.CustomValidator{
					Imports: []code.Import{
						{Path: "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"},
					},
					SchemaDefinition: schemaDefinition,
				},
			})
		}
	} else if validation != "" {
		// It's a regex pattern - translate to appropriate validator
		translatedValidators := g.translateRegexToValidators(validation)
		validators = append(validators, translatedValidators...)
	}

	return validators
}

// translateRegexToValidators converts a regex pattern to appropriate Terraform validators.
func (g *SpecificationGenerator) translateRegexToValidators(pattern string) []schema.StringValidator {
	validators := make([]schema.StringValidator, 0)

	// Try to translate to specific validators
	if validator := g.tryLengthValidator(pattern); validator != nil {
		validators = append(validators, *validator)
		return validators
	}

	if validator := g.tryHexValidator(pattern); validator != nil {
		validators = append(validators, *validator)
		return validators
	}

	if validator := g.tryColorHexValidator(pattern); validator != nil {
		validators = append(validators, *validator)
		return validators
	}

	// For complex patterns, use RegexMatches but with proper escaping
	// Replace problematic characters to avoid escape sequence issues
	safePattern := strings.ReplaceAll(pattern, `"`, `'`)
	schemaDefinition := fmt.Sprintf("stringvalidator.RegexMatches(regexp.MustCompile(`%s`), \"\")", safePattern)
	validators = append(validators, schema.StringValidator{
		Custom: &schema.CustomValidator{
			Imports: []code.Import{
				{Path: "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"},
				{Path: "regexp"},
			},
			SchemaDefinition: schemaDefinition,
		},
	})

	return validators
}

// tryLengthValidator attempts to create a length-based validator from a regex pattern.
func (g *SpecificationGenerator) tryLengthValidator(pattern string) *schema.StringValidator {
	// Pattern: .{min,max} or .{exact} or .{min,}
	lengthPattern := regexp.MustCompile(`^\.\{(\d+)(?:,(\d*))?\}$`)
	matches := lengthPattern.FindStringSubmatch(pattern)

	if len(matches) > 0 {
		min := matches[1] //nolint:predeclared
		max := matches[2] //nolint:predeclared
		hasComma := strings.Contains(pattern, ",")

		var schemaDefinition string

		if !hasComma {
			// Exact length: .{n} (no comma in pattern)
			schemaDefinition = fmt.Sprintf("stringvalidator.LengthBetween(%s, %s)", min, min)
		} else if max == "" {
			// Minimum length: .{min,} (comma but no max)
			minInt, _ := strconv.Atoi(min)
			schemaDefinition = fmt.Sprintf("stringvalidator.LengthAtLeast(%d)", minInt)
		} else {
			// Range: .{min,max}
			minInt, _ := strconv.Atoi(min)
			maxInt, _ := strconv.Atoi(max)
			schemaDefinition = fmt.Sprintf("stringvalidator.LengthBetween(%d, %d)", minInt, maxInt)
		}

		return &schema.StringValidator{
			Custom: &schema.CustomValidator{
				Imports: []code.Import{
					{Path: "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"},
				},
				SchemaDefinition: schemaDefinition,
			},
		}
	}

	return nil
}

// tryHexValidator attempts to create a validator for hexadecimal patterns.
func (g *SpecificationGenerator) tryHexValidator(pattern string) *schema.StringValidator {
	// Pattern: [0-9A-Fa-f]{n} or [0-9a-fA-F]{n}
	hexPattern := regexp.MustCompile(`^\[0-9A-Fa-f\]\{(\d+)\}$|^\[0-9a-fA-F\]\{(\d+)\}$`)
	matches := hexPattern.FindStringSubmatch(pattern)

	if len(matches) > 0 {
		length := matches[1]
		if length == "" {
			length = matches[2]
		}

		lengthInt, _ := strconv.Atoi(length)

		// Use length validator combined with regex for hex characters
		schemaDefinition := fmt.Sprintf("stringvalidator.LengthBetween(%d, %d)", lengthInt, lengthInt)

		return &schema.StringValidator{
			Custom: &schema.CustomValidator{
				Imports: []code.Import{
					{Path: "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"},
				},
				SchemaDefinition: schemaDefinition,
			},
		}
	}

	return nil
}

// tryColorHexValidator attempts to create a validator for color hex codes.
func (g *SpecificationGenerator) tryColorHexValidator(pattern string) *schema.StringValidator {
	// Pattern: ^#(?:[0-9a-fA-F]{3}){1,2}$
	if strings.Contains(pattern, "#") && strings.Contains(pattern, "[0-9a-fA-F]{3}") {
		// This is a color hex pattern - use LengthBetween for #RGB or #RRGGBB
		schemaDefinition := "stringvalidator.LengthBetween(4, 7)" // #RGB (4) to #RRGGBB (7)

		return &schema.StringValidator{
			Custom: &schema.CustomValidator{
				Imports: []code.Import{
					{Path: "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"},
				},
				SchemaDefinition: schemaDefinition,
			},
		}
	}

	return nil
}

// buildInt64Validators creates int64 validators from validation string.
func (g *SpecificationGenerator) buildInt64Validators(validation string) []schema.Int64Validator {
	if validation == "" {
		return nil
	}

	validators := make([]schema.Int64Validator, 0)

	// Check if it's a pipe-separated list of values
	if strings.Contains(validation, "|") {
		values := strings.Split(validation, "|")
		if len(values) > 1 {
			// Try to parse as integers
			intValues := make([]string, 0)
			for _, v := range values {
				v = strings.TrimSpace(v)
				if _, err := strconv.ParseInt(v, 10, 64); err == nil {
					intValues = append(intValues, v)
				}
			}
			if len(intValues) > 0 {
				schemaDefinition := fmt.Sprintf("int64validator.OneOf(%s)", strings.Join(intValues, ", "))
				validators = append(validators, schema.Int64Validator{
					Custom: &schema.CustomValidator{
						Imports: []code.Import{
							{Path: "github.com/hashicorp/terraform-plugin-framework-validators/int64validator"},
						},
						SchemaDefinition: schemaDefinition,
					},
				})
			}
		}
	}

	return validators
}

// buildFloat64Validators creates float64 validators from validation string.
func (g *SpecificationGenerator) buildFloat64Validators(_ string) []schema.Float64Validator { //nolint:unused
	// For now, float64 validators are less common, return nil
	return nil
}

// fieldTypeToElementType converts a Go type to an ElementType.
func (g *SpecificationGenerator) fieldTypeToElementType(fieldType string) schema.ElementType {
	switch fieldType {
	case "bool":
		return schema.ElementType{Bool: &schema.BoolType{}}
	case fields.Int:
		return schema.ElementType{Int64: &schema.Int64Type{}}
	case "float64":
		return schema.ElementType{Float64: &schema.Float64Type{}}
	case "string":
		return schema.ElementType{String: &schema.StringType{}}
	default:
		// Default to string
		return schema.ElementType{String: &schema.StringType{}}
	}
}

// toTerraformName converts a Go struct name to a Terraform resource/data source name.
func toTerraformName(name string) string {
	// Convert CamelCase to snake_case and lowercase
	return strings.ToLower(strcase.ToSnake(name))
}

// WriteSpecification writes the specification to a JSON file.
func (g *SpecificationGenerator) WriteSpecification(outputPath string) error {
	spec := g.Generate()

	if err := spec.Validate(context.Background()); err != nil {
		panic(err)
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal specification: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write specification file: %w", err)
	}

	return nil
}

func ptr[T any](in T) *T {
	return &in
}

func findMembers(a resource.Attribute) bool {
	return a.Name == "members"
}

func findConfigNetwork(a resource.Attribute) bool {
	return a.Name == "config_network"
}

func findAttr(name string) func(a resource.Attribute) bool {
	return func(a resource.Attribute) bool {
		return a.Name == name
	}
}
