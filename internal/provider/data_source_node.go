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

func dataSourceNode() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Node data source in k3d.",

		ReadContext: dataSourceNodeRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Node name.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"cluster": {
				Description: "Select the cluster that the node shall connect to.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"role": {
				Description: "Specify node role [server, agent].",
				Computed:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func dataSourceNodeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nodeName := d.Get("name").(string)
	nodeID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, nodeName)
	d.SetId(nodeID)

	node, err := client.NodeGet(ctx, runtimes.SelectedRuntime, &types.Node{Name: nodeID})
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("cluster", node.K3sNodeLabels[types.LabelClusterName]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("role", string(node.Role)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
