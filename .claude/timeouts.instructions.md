---
name: Timeouts Instruction
description: This file describes the implementation of timeouts for resources and data sources in the provider using the terraform-plugin-framework-timeouts Go module.
applyTo: "**/*_{resource,data_source,ephemeral,action}.go"
---


## Terraform Plugin Framework Timeouts

Reference the repo here: https://github.com/hashicorp/terraform-plugin-framework-timeouts

Implement timeouts for all resources and data sources in the provider using the terraform-plugin-framework-timeouts Go module. This will allow practitioners to specify timeouts for operations in their Terraform configuration, and for the provider to use those timeouts when performing operations.

terraform-plugin-framework-timeouts is a Go module containing convenience functions and types for timeouts for use with [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework). It aims to provide simple access and usage of timeouts defined within configuration.

### Terraform Plugin Framework Compatibility

This Go module is typically kept up to date with the latest `terraform-plugin-framework` releases to ensure all timeouts functionality is available.

### Go Compatibility

This Go module follows `terraform-plugin-framework` Go compatibility.

Currently, that means Go **1.25** must be used when developing and testing code.

### Usage

Usage of this module requires the following changes in the provider code:

- [Schema Mutation](#schema-mutation)
- [Updating Models](#updating-models)
- [Accessing Timeouts in CRUD Functions](#accessing-timeouts-in-crud-functions)

#### Schema Mutation

Timeouts can be defined using either nested blocks or nested attributes.

If you are writing a new provider using [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework)
then we recommend using nested attributes.

If you are [migrating a provider from SDKv2 to the Framework](https://www.terraform.io/plugin/framework/migrating) and 
you are already using timeouts you can either continue to use block syntax, or switch to using nested attributes. 
However, switching to using nested attributes will require that practitioners that are using your provider update their
Terraform configuration.

###### Block

The following illustrates nested block syntax for defining timeouts on a resource and a data source.

```terraform
resource "timeouts_example" "example" {
  /* ... */

  timeouts {
    create = "60m"
  }
}
```

```terraform
data "timeouts_example" "example" {
  /* ... */

  timeouts {
    read = "30m"
  }
}
```

Use this module to mutate the `schema.Schema`:

You must supply `timeouts.Opts` when calling `timeouts.Block()` on a resource. The supplied `timeouts.Opts` allows specifying which timeouts to create and whether to override the default description for the timeout.

Alternatively, `timeouts.BlockAll()` will generate attributes for `create`, `read`, `update` and `delete`.

```go
import (
    /* ... */
    "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
)

func (t *exampleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        /* ... */
        
        Blocks: map[string]schema.Block{
            "timeouts": timeouts.Block(ctx,
                timeouts.Opts{
                    Create: true,
                },
             )
        },
```

The `timeouts.Block()` call does not accept options on a data source as `read` is the only option.

However, the `timeouts.BlockWithOpts()` function is available for overriding the default description.

```go
import (
    /* ... */
    "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
)

func (t exampleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        /* ... */

        Blocks: map[string]schema.Block{
            "timeouts": timeouts.Block(ctx),
        },
    }
}
```

###### Attribute 

The following illustrates nested attribute syntax for defining timeouts on a resource and a data source.

```terraform
resource "timeouts_example" "example" {
  /* ... */

  timeouts = {
    create = "60m"
  }
}
```

```terraform
data "timeouts_example" "example" {
  /* ... */

  timeouts = {
    read = "30m"
  }
}
```

Use this module to mutate the `schema.Schema` as follows:

You must supply `timeouts.Opts` when calling `timeouts.Attributes()` on a resource. The supplied `timeouts.Opts` allows specifying which timeouts to create and whether to override the default description for the timeout.

Alternatively, `timeouts.AttributesAll()` will generate attributes for `create`, `read`, `update` and `delete`.

```go
import (
    /* ... */
    "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
)

func (t *exampleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            /* ... */
            "timeouts": timeouts.Attributes(ctx, timeouts.Opts{
                Create: true,
            }),
        },
```

The `timeouts.Attributes()` call does not accept options on a data source as `read` is the only option. 

However, the `timeouts.AttributesWithOpts()` function is available for overriding the default description.

```go
import (
    /* ... */
    "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
)

func (t exampleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            /* ... */
            "timeouts": timeouts.Attributes(ctx),
        },
    }
}
```

#### Updating Models

In functions in which the config, state or plan is being unmarshalled, for instance, the `Create` function:

```go
func (r exampleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data exampleResourceData

    diags := req.Plan.Get(ctx, &data)
    resp.Diagnostics.Append(diags...)
```

The model that is being used, `exampleResourceData` in this example, will need to be modified to include a field for
timeouts which is of type `timeouts.Value`. For example:

```go
type exampleResourceData struct {
    /* ... */
    Timeouts    timeouts.Value `tfsdk:"timeouts"`
```

#### Accessing Timeouts in CRUD Functions

Once the model has been populated with the config, state or plan the duration of the timeout can be accessed by calling
the appropriate helper function and then used to configure timeout behaviour, for instance:

```go
func (r exampleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data exampleResourceData

    diags := req.Plan.Get(ctx, &data)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
	
    // Create() is passed a default timeout to use if no value 
    // has been supplied in the Terraform configuration.
    createTimeout, err := data.Timeouts.Create(ctx, 20*time.Minute)
    if err != nil {
        // handle error
    }
    
    ctx, cancel := context.WithTimeout(ctx, createTimeout)
    defer cancel()
    
    /* ... */
}
```