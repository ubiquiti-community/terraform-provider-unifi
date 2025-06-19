package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataAccount() *schema.Resource {
	return &schema.Resource{
		Description: "`unifi_account` data source can be used to retrieve RADIUS user accounts",

		ReadContext: dataAccountRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of this account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"site": {
				Description: "The name of the site the account is associated with.",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"name": {
				Description: "The name of the account to look up",
				Type:        schema.TypeString,
				Required:    true,
			},

			"password": {
				Description: "The password of the account.",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
			"tunnel_type": {
				Description: "See RFC2868 section 3.1", // @TODO: better documentation https://help.ui.com/hc/en-us/articles/360015268353-UniFi-USG-UDM-Configuring-RADIUS-Server#6
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"tunnel_medium_type": {
				Description: "See RFC2868 section 3.2", // @TODO: better documentation https://help.ui.com/hc/en-us/articles/360015268353-UniFi-USG-UDM-Configuring-RADIUS-Server#6
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"network_id": {
				Description: "ID of the network for this account",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataAccountRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c, ok := meta.(*client)
	if !ok {
		return diag.Errorf("meta is not of type *client")
	}

	name, ok := d.Get("name").(string)
	if !ok {
		return diag.Errorf("name is not a string")
	}
	site, ok := d.Get("site").(string)
	if !ok {
		return diag.Errorf("site is not a string")
	}
	if site == "" {
		site = c.site
	}

	accounts, err := c.c.ListAccounts(ctx, site)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, account := range accounts {
		if account.Name == name {
			d.SetId(account.ID)
			_ = d.Set("name", account.Name)
			_ = d.Set("password", account.XPassword)
			_ = d.Set("tunnel_type", account.TunnelType)
			_ = d.Set("tunnel_medium_type", account.TunnelMediumType)
			_ = d.Set("network_id", account.NetworkID)
			_ = d.Set("site", site)
			return nil
		}
	}

	return diag.Errorf("Account not found with name %s", name)
}
