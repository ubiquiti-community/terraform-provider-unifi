---
description: 'Terraform plugin framework migration guide.'
applyTo: '**.go'
---
## Style Guide for Terraform Plugin Framework Migration

Refer to the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/docs/plugin-framework) documentation for detailed guidelines.

Use the styleguide.md file as a reference for best practices and conventions when migrating to the Terraform Plugin Framework.

This document outlines the key principles and practices to follow during the migration process.

## Directory and File Structure

- **Package organization**: Avoid package sprawl; group related functionality appropriately
- **Utility packages**: Avoid generic "util" packages - use descriptive names (e.g., `wait.Poll` instead of `util.Poll`)
- **Naming conventions**: 
  - All filenames lowercase
  - Use underscores in Go source files/directories (not dashes)
  - Package directories avoid separators when possible
  - Use nested subdirectories for multi-word package names

## Project Structure

- **Directory layout**: `{provider}/` for main code, `util/` for type conversion, `testhelper/` for test utilities
- **File naming**: `resource_{name}.go`, `data_source_{name}.go` for clear identification
- **Essential imports**: Always include context, framework types, validators, and plan modifiers
- **Interface compliance**: Declare interface satisfaction with compile-time checks:
  ```go
  var (
      _ resource.Resource                = &myResource{}
      _ resource.ResourceWithConfigure   = &myResource{}
      _ resource.ResourceWithImportState = &myResource{}
  )
  ```

## Resource Model Design
- **Struct tags**: Use `tfsdk:"field_name"` tags matching schema attribute names
- **Type selection**: `types.String`, `types.List`, `types.Dynamic` for complex data
- **Required vs Optional vs Computed**: Match API behavior - required for user input, computed for API-generated values
- **Null handling**: Always use null types (`types.StringNull()`) for empty/unset values

## Schema Definition Patterns
- **Field attributes**: 
  - `Required: true` for mandatory user input
  - `Optional: true, Computed: true` for fields that can be set by user OR API
  - `Computed: true` for read-only API values
- **Validation**: Use framework validators (`stringvalidator.ConflictsWith`, `listvalidator.SizeAtLeast`)
- **Plan modifiers**: `RequiresReplace()` only for truly immutable fields
- **Conflict handling**: Use `ConflictsWith(path.MatchRoot("other_field"))` for mutually exclusive options

## State Management (Critical for Import Success)
- **Null vs Empty**: Always use `types.StringNull()` for empty strings, `types.ListNull()` for empty lists
- **Read function consistency**: Must handle API responses identically in Create and Read operations
- **Import verification**: Test with `ImportStateVerify: true` to catch state mismatches
- **Critical pattern for strings**:
  ```go
  if apiResponse.Field == "" {
      model.Field = types.StringNull()
  } else {
      model.Field = types.StringValue(apiResponse.Field)
  }
  ```
- **Critical pattern for lists**:
  ```go
  if len(apiResponse.Items) > 0 {
      // Create list with values
  } else {
      model.Items = types.ListNull(types.StringType)
  }
  ```

## Type Conversion Best Practices
- **Dynamic types**: Check `IsNull()` and `IsUnknown()` before conversion
- **Empty maps**: Use `len(input) > 0` check, return `types.DynamicNull()` for empty
- **List extraction**: Use `ElementsAs(ctx, &slice, false)` to convert to Go slices
- **Map to Dynamic**: Create utility functions for consistent conversion patterns
- **Go slice to List**: Pre-allocate slice with `make([]attr.Value, len(source))`

## Error Handling Patterns
- **Attribute errors**: Use `AddAttributeError(path.Root("field"), title, detail)` for field-specific issues
- **API errors**: Check specific HTTP codes with `gophercloud.ResponseCodeIs(err, http.StatusNotFound)`
- **Diagnostics flow**: Always check `resp.Diagnostics.HasError()` before continuing
- **Resource cleanup**: Delete created resources in Create function if post-creation steps fail

## Resource Model Design
- **Struct tags**: Use `tfsdk:"field_name"` tags matching schema attribute names
- **Type selection**: `types.String`, `types.List`, `types.Dynamic` for complex data
- **Required vs Optional vs Computed**: Match API behavior - required for user input, computed for API-generated values
- **Null handling**: Always use null types (`types.StringNull()`) for empty/unset values

## Schema Definition Patterns
- **Field attributes**: 
  - `Required: true` for mandatory user input
  - `Optional: true, Computed: true` for fields that can be set by user OR API
  - `Computed: true` for read-only API values
- **Validation**: Use framework validators (`stringvalidator.ConflictsWith`, `listvalidator.SizeAtLeast`)
- **Plan modifiers**: `RequiresReplace()` only for truly immutable fields
- **Conflict handling**: Use `ConflictsWith(path.MatchRoot("other_field"))` for mutually exclusive options

## State Management (Critical for Import Success)
- **Null vs Empty**: Always use `types.StringNull()` for empty strings, `types.ListNull()` for empty lists
- **Read function consistency**: Must handle API responses identically in Create and Read operations
- **Import verification**: Test with `ImportStateVerify: true` to catch state mismatches
- **Critical pattern for strings**:
  ```go
  if apiResponse.Field == "" {
      model.Field = types.StringNull()
  } else {
      model.Field = types.StringValue(apiResponse.Field)
  }
  ```
- **Critical pattern for lists**:
  ```go
  if len(apiResponse.Items) > 0 {
      // Create list with values
  } else {
      model.Items = types.ListNull(types.StringType)
  }
  ```

## Type Conversion Best Practices
- **Dynamic types**: Check `IsNull()` and `IsUnknown()` before conversion
- **Empty maps**: Use `len(input) > 0` check, return `types.DynamicNull()` for empty
- **List extraction**: Use `ElementsAs(ctx, &slice, false)` to convert to Go slices
- **Map to Dynamic**: Create utility functions for consistent conversion patterns
- **Go slice to List**: Pre-allocate slice with `make([]attr.Value, len(source))`

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

## Code Organization

- **New libraries**: Place in `pkg/util` subdirectories if no better home exists
- **Third-party code**: Manage Go dependencies with modules; other code goes in `third_party/`

## Testing Requirements

- **Unit tests**: Required for all new packages and significant functionality
- **Test style**: Prefer table-driven tests for multiple scenarios
- **Integration tests**: Required for significant features and kubectl commands
- **Cross-platform**: Tests must pass on macOS and Windows
- **Async testing**: Use wait/retry patterns instead of fixed delays
- **Dependencies**: Use Google Cloud Artifact Registry instead of Docker Hub

## Key Principles

- Prioritize descriptive naming over generic utilities
- Ensure cross-platform compatibility
- Write comprehensive tests at multiple levels
- Follow established patterns for package organization
```