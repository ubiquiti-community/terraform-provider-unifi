package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataDNSRecord() *schema.Resource {
	return &schema.Resource{
		Description: "`unifi_dns_record` data source can be used to retrieve the ID for an DNS record by name.",

		ReadContext: dataDNSRecordRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of this DNS record.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"site": {
				Description: "The name of the site the DNS record is associated with.",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"name": {
				Description: "The name of the DNS record to look up, leave blank to look up the default DNS record.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"port": {
				Description: "The port of the DNS record.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"priority": {
				Description: "The priority of the DNS record.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"record_type": {
				Description: "The type of the DNS record.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"ttl": {
				Description: "The TTL of the DNS record.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"value": {
				Description: "The value of the DNS record.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"weight": {
				Description: "The weight of the DNS record.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
		},
	}
}

func dataDNSRecordRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
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

	groups, err := c.c.ListDNSRecord(ctx, site)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, g := range groups {
		if (name == "" && g.HiddenID == "default") || g.Key == name {
			d.SetId(g.ID)
			_ = d.Set("site", site)
			_ = d.Set("port", g.Port)
			_ = d.Set("priority", g.Priority)
			_ = d.Set("record_type", g.RecordType)
			_ = d.Set("ttl", g.Ttl)
			_ = d.Set("value", g.Value)
			_ = d.Set("weight", g.Weight)

			return nil
		}
	}

	return diag.Errorf("DNS record not found with name %s", name)
}
