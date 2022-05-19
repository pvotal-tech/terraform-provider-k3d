package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
)

func dataSourceRegistry() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "k3d-managed registry.",

		ReadContext: dataSourceRegistryRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Registry name.",
				Required:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func dataSourceRegistryRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	registryName := d.Get("name").(string)
	registryID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, registryName)
	d.SetId(registryID)

	_, err := client.NodeGet(ctx, runtimes.SelectedRuntime, &types.Node{Name: registryID})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
