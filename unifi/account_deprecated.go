package unifi

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// deprecatedAccountResource wraps radiusUserResource to provide the old
// "unifi_account" resource type name as a deprecated alias. This avoids a
// breaking change for users who already have unifi_account in their state.
type deprecatedAccountResource struct {
	radiusUserResource
}

func NewDeprecatedAccountResource() resource.Resource {
	return &deprecatedAccountResource{}
}

func (r *deprecatedAccountResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (r *deprecatedAccountResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	// Get the base schema from the real resource
	r.radiusUserResource.Schema(ctx, req, resp)

	// Add deprecation message
	resp.Schema.DeprecationMessage = "Use unifi_radius_user instead. This resource will be removed in a future version."
}

// deprecatedAccountDataSource wraps radiusUserDataSource to provide the old
// "unifi_account" data source type name as a deprecated alias.
type deprecatedAccountDataSource struct {
	radiusUserDataSource
}

func NewDeprecatedAccountDataSource() datasource.DataSource {
	return &deprecatedAccountDataSource{}
}

func (d *deprecatedAccountDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *deprecatedAccountDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	// Get the base schema from the real data source
	d.radiusUserDataSource.Schema(ctx, req, resp)

	// Add deprecation message
	resp.Schema.DeprecationMessage = "Use the unifi_radius_user data source instead. This data source will be removed in a future version."
}
