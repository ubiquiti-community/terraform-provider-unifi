# Terraform Plugin Framework Development Guide

A comprehensive guide for developing Terraform providers using the terraform-plugin-framework, based on real-world experience with complex resource management scenarios.

## Table of Contents

1. [Project Structure & Setup](#project-structure--setup)
2. [Resource Model Design](#resource-model-design)  
3. [Schema Definition Best Practices](#schema-definition-best-practices)
4. [State Management & Import](#state-management--import)
5. [Type Conversion & Dynamic Data](#type-conversion--dynamic-data)
6. [Validation & Plan Modifiers](#validation--plan-modifiers)
7. [Error Handling](#error-handling)
8. [Testing Strategies](#testing-strategies)
9. [Common Pitfalls](#common-pitfalls)

## Project Structure & Setup

### Directory Layout
```
terraform-provider-{name}/
├── main.go                          # Provider entry point
├── {provider}/                      # Provider-specific code
│   ├── provider.go                  # Provider definition
│   ├── clients.go                   # API client management
│   ├── resource_{name}.go           # Resource implementations
│   ├── data_source_{name}.go        # Data source implementations
│   └── util/                        # Utility functions
│       ├── dynamic.go               # Type conversion utilities
│       └── update_opt.go            # API update option helpers
├── docs/                            # Auto-generated documentation
├── examples/                        # Example configurations
└── testhelper/                      # Test utilities
```

### Essential Imports
```go
import (
    "context"
    "fmt"
    
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-framework/diag"
    "github.com/hashicorp/terraform-plugin-framework/path"
    
    // Validators
    "github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
    "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
    
    // Plan modifiers
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
)
```

## Resource Model Design

### Model Structure
```go
type resourceModel struct {
    ID             types.String  `tfsdk:"id"`
    Name           types.String  `tfsdk:"name"`
    RequiredField  types.String  `tfsdk:"required_field"`
    OptionalList   types.List    `tfsdk:"optional_list"`
    ComputedField  types.String  `tfsdk:"computed_field"`
    DynamicData    types.Dynamic `tfsdk:"dynamic_data"`
}
```

### Interface Implementation
```go
// Ensure your resource satisfies required interfaces
var (
    _ resource.Resource                = &myResource{}
    _ resource.ResourceWithConfigure   = &myResource{}
    _ resource.ResourceWithImportState = &myResource{}
)
```

## Schema Definition Best Practices

### Field Types & Attributes
```go
func (r *myResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Manages a resource with comprehensive examples.",
        Attributes: map[string]schema.Attribute{
            // Required fields
            "required_field": schema.StringAttribute{
                MarkdownDescription: "A required string field.",
                Required:            true,
            },
            
            // Optional fields with validation
            "optional_field": schema.StringAttribute{
                MarkdownDescription: "An optional field with validation.",
                Optional:            true,
                Validators: []validator.String{
                    stringvalidator.LengthAtLeast(1),
                },
            },
            
            // Computed fields
            "computed_field": schema.StringAttribute{
                MarkdownDescription: "A computed field set by the API.",
                Computed:            true,
            },
            
            // Optional + Computed (state can be set by user OR API)
            "optional_computed": schema.StringAttribute{
                MarkdownDescription: "Field that can be set by user or computed.",
                Optional:            true,
                Computed:            true,
            },
            
            // Lists with validation
            "list_field": schema.ListAttribute{
                MarkdownDescription: "A list of strings with conflict validation.",
                ElementType:         types.StringType,
                Optional:            true,
                Computed:            true,
                Validators: []validator.List{
                    listvalidator.ConflictsWith(path.MatchRoot("other_field")),
                },
                PlanModifiers: []planmodifier.List{
                    listplanmodifier.RequiresReplace(),
                },
            },
            
            // Dynamic fields for flexible data
            "extra": schema.DynamicAttribute{
                MarkdownDescription: "Extra metadata as key-value pairs.",
                Optional:            true,
                PlanModifiers: []planmodifier.Dynamic{
                    dynamicplanmodifier.RequiresReplace(),
                },
            },
        },
    }
}
```

### Plan Modifiers
- **RequiresReplace**: Forces resource recreation when field changes
- **UseStateForUnknown**: Preserves current state value when new value is unknown
- **RequiresReplaceIf**: Conditional replacement based on custom logic

## State Management & Import

### Critical: Consistent Null Handling
The most common cause of import issues is inconsistent handling of null vs empty values.

```go
func (r *myResource) readResourceData(ctx context.Context, model *resourceModel, diagnostics *diag.Diagnostics) {
    // ❌ WRONG: Always setting string values
    model.LastError = types.StringValue(apiResponse.LastError)
    
    // ✅ CORRECT: Handle empty strings as null
    if apiResponse.LastError == "" {
        model.LastError = types.StringNull()
    } else {
        model.LastError = types.StringValue(apiResponse.LastError)
    }
    
    // ❌ WRONG: Setting empty lists as values
    if len(apiResponse.Items) > 0 {
        // ... populate list
    } else {
        model.Items = types.ListValueMust(types.StringType, []attr.Value{})
    }
    
    // ✅ CORRECT: Use null for empty lists
    if len(apiResponse.Items) > 0 {
        // ... populate list
    } else {
        model.Items = types.ListNull(types.StringType)
    }
}
```

### Import State Implementation
```go
func (r *myResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Pass through the ID for simple imports
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
    
    // For complex imports, parse the import identifier
    // Example: "project/resource_id" format
    parts := strings.Split(req.ID, "/")
    if len(parts) != 2 {
        resp.Diagnostics.AddError(
            "Invalid Import ID",
            "Import ID must be in format 'project/resource_id'",
        )
        return
    }
    
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
```

### Read Function Best Practices
```go
func (r *myResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state resourceModel
    
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    
    // Use a helper function for consistency
    r.readResourceData(ctx, &state, &resp.Diagnostics)
    if resp.Diagnostics.HasError() {
        return
    }
    
    // Always set the state back
    resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
```

## Type Conversion & Dynamic Data

### Dynamic Type Handling
```go
// Utility for converting maps to Dynamic types
func StringMapToDynamic(ctx context.Context, input map[string]string) (types.Dynamic, error) {
    if input == nil {
        return types.DynamicNull(), nil
    }
    
    if len(input) == 0 {
        // Empty map - create an empty object
        objectType := types.ObjectType{AttrTypes: map[string]attr.Type{}}
        objectValue := types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{})
        return types.DynamicValue(objectValue), nil
    }
    
    // Convert all values to string types
    attrTypes := make(map[string]attr.Type)
    attrValues := make(map[string]attr.Value)
    
    for key, value := range input {
        attrTypes[key] = types.StringType
        attrValues[key] = types.StringValue(value)
    }
    
    objectValue, diags := types.ObjectValue(attrTypes, attrValues)
    if diags.HasError() {
        return types.DynamicNull(), fmt.Errorf("error creating object: %v", diags)
    }
    
    return types.DynamicValue(objectValue), nil
}

// Converting Dynamic back to Go types
func DynamicToStringMap(ctx context.Context, dynamic types.Dynamic) (map[string]string, error) {
    if dynamic.IsNull() || dynamic.IsUnknown() {
        return nil, nil
    }
    
    // Handle the conversion based on underlying type
    // Implementation depends on your specific needs
}
```

### List Handling
```go
// Creating lists from API responses
if len(apiResponse.Items) > 0 {
    itemValues := make([]attr.Value, len(apiResponse.Items))
    for i, item := range apiResponse.Items {
        itemValues[i] = types.StringValue(item)
    }
    itemsList, diags := types.ListValue(types.StringType, itemValues)
    diagnostics.Append(diags...)
    if diagnostics.HasError() {
        return
    }
    model.Items = itemsList
} else {
    model.Items = types.ListNull(types.StringType)
}

// Extracting lists to Go slices
if !plan.Items.IsNull() && !plan.Items.IsUnknown() {
    var items []string
    diags := plan.Items.ElementsAs(ctx, &items, false)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    // Use items slice...
}
```

## Validation & Plan Modifiers

### Custom Validators
```go
// Conflict validation
schema.StringAttribute{
    Validators: []validator.String{
        stringvalidator.ConflictsWith(path.MatchRoot("other_field")),
    },
}

// Length validation
schema.StringAttribute{
    Validators: []validator.String{
        stringvalidator.LengthBetween(1, 255),
    },
}

// List validation
schema.ListAttribute{
    Validators: []validator.List{
        listvalidator.SizeAtLeast(1),
        listvalidator.ConflictsWith(path.MatchRoot("alternative_field")),
    },
}
```

### Plan Modifiers for Immutable Resources
```go
// Fields that require replacement when changed
"immutable_field": schema.StringAttribute{
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),
    },
},

// Lists that require replacement
"immutable_list": schema.ListAttribute{
    PlanModifiers: []planmodifier.List{
        listplanmodifier.RequiresReplace(),
    },
},
```

## Error Handling

### Structured Error Reporting
```go
// Attribute-specific errors
resp.Diagnostics.AddAttributeError(
    path.Root("field_name"),
    "Error Title",
    "Detailed error message explaining what went wrong and how to fix it.",
)

// General errors
resp.Diagnostics.AddError(
    "API Error",
    fmt.Sprintf("Could not perform operation: %s", err),
)

// Warning (non-blocking)
resp.Diagnostics.AddWarning(
    "Deprecation Warning",
    "This field is deprecated and will be removed in a future version.",
)
```

### API Error Handling
```go
// Check for specific HTTP status codes
if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
    // Resource doesn't exist - this is OK for delete operations
    return
}

// Handle different error types
switch e := err.(type) {
case gophercloud.ErrDefault404:
    // Handle 404 specifically
case gophercloud.ErrDefault409:
    // Handle conflict errors
default:
    // Generic error handling
}
```

## Testing Strategies

### Acceptance Tests
```go
func TestAccResource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV5ProviderFactories: protoV5ProviderFactories(),
        CheckDestroy:             testAccResourceDestroy,
        Steps: []resource.TestStep{
            {
                Config: testAccResourceConfig_basic(),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("myresource.test", "name", "test"),
                    resource.TestCheckResourceAttrSet("myresource.test", "id"),
                ),
            },
        },
    })
}

// Test imports specifically
func TestAccResource_import(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV5ProviderFactories: protoV5ProviderFactories(),
        CheckDestroy:             testAccResourceDestroy,
        Steps: []resource.TestStep{
            {
                Config: testAccResourceConfig_basic(),
            },
            {
                ResourceName:      "myresource.test",
                ImportState:       true,
                ImportStateVerify: true,
                // ImportStateVerifyIgnore: []string{"field_not_in_api"},
            },
        },
    })
}
```

## Common Pitfalls

### 1. State Mismatch After Import
**Problem**: Plan shows changes after importing a resource.

**Solution**:
- Ensure consistent null handling in read functions
- Match computed field behavior between create and read
- Handle empty vs null values consistently

```go
// ❌ Inconsistent handling
model.Field = types.StringValue(apiResponse.Field) // Always sets a value

// ✅ Consistent handling  
if apiResponse.Field == "" {
    model.Field = types.StringNull()
} else {
    model.Field = types.StringValue(apiResponse.Field)
}
```

### 2. Type Conversion Errors
**Problem**: Dynamic types failing to convert properly.

**Solution**:
- Always check for null/unknown before conversion
- Handle empty collections as null, not empty values
- Use utility functions for consistent conversion

### 3. Plan Modifier Misuse
**Problem**: Unexpected replacement or diff behavior.

**Solution**:
- Use `RequiresReplace` sparingly, only for truly immutable fields
- Test plan behavior thoroughly with imports
- Consider `UseStateForUnknown` for computed fields

### 4. Validation Conflicts
**Problem**: Validators preventing valid configurations.

**Solution**:
- Test edge cases thoroughly
- Use `ConflictsWith` carefully - ensure mutual exclusivity is correct
- Document validation requirements clearly

### 5. Concurrent Modification Issues
**Problem**: Resources getting modified during Terraform operations.

**Solution**:
- Implement proper error handling for concurrent modifications
- Use appropriate timeouts for long-running operations
- Consider implementing retry logic for transient failures

## Advanced Patterns

### Wait Functions for Async Resources
```go
func (r *myResource) waitForCompletion(ctx context.Context, id string, model *resourceModel, diagnostics *diag.Diagnostics) error {
    timeout := 5 * time.Minute
    checkInterval := 10 * time.Second
    
    for {
        r.readResourceData(ctx, model, diagnostics)
        if diagnostics.HasError() {
            return fmt.Errorf("error reading resource during wait")
        }
        
        state := model.State.ValueString()
        switch state {
        case "active", "complete":
            return nil
        case "error", "failed":
            return fmt.Errorf("resource entered error state: %s", model.LastError.ValueString())
        default:
            time.Sleep(checkInterval)
            timeout -= checkInterval
            if timeout <= 0 {
                return fmt.Errorf("timeout waiting for resource completion")
            }
        }
    }
}
```

### Conditional Logic in CRUD Operations
```go
func (r *myResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan resourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    
    // Handle mutually exclusive fields
    if !plan.FieldA.IsNull() && !plan.FieldB.IsNull() {
        resp.Diagnostics.AddError(
            "Conflicting Configuration",
            "Cannot specify both field_a and field_b",
        )
        return
    }
    
    // Build API request based on what's provided
    createOpts := &CreateOpts{}
    
    if !plan.FieldA.IsNull() {
        createOpts.FieldA = plan.FieldA.ValueString()
    } else if !plan.FieldB.IsNull() {
        createOpts.FieldB = plan.FieldB.ValueString()  
    } else {
        resp.Diagnostics.AddError(
            "Missing Required Field",
            "Must specify either field_a or field_b",
        )
        return
    }
    
    // Continue with creation...
}
```

