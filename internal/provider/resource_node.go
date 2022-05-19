package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/version"
)

func resourceNode() *schema.Resource {
	k3dVersion, err := version.GetK3sVersion("stable")
	if err != nil {
		panic(err)
	}
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Containerized k3s node (k3s in docker).",

		CreateContext: resourceNodeCreate,
		ReadContext:   resourceNodeRead,
		// UpdateContext: resourceNodeUpdate,
		DeleteContext: resourceNodeDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Node name.",
				ForceNew:    true,
				Required:    true,
				Type:        schema.TypeString,
			},
			"cluster": {
				Description: "Select the cluster that the node shall connect to.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeString,
				Default:     types.DefaultClusterName,
			},
			"image": {
				Description: "Specify k3s image used for the node(s).",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeString,
				Default:     fmt.Sprintf("%s:%s", types.DefaultK3sImageRepo, k3dVersion),
			},
			"memory": {
				Description: "Memory limit imposed on the node [From docker]",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeString,
			},
			"role": {
				Description:  "Specify node role [server, agent].",
				ForceNew:     true,
				Optional:     true,
				Type:         schema.TypeString,
				Default:      string(types.AgentRole),
				ValidateFunc: validation.StringInSlice([]string{string(types.AgentRole), string(types.ServerRole)}, true),
			},
		},
	}
}

func resourceNodeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterName := d.Get("cluster").(string)
	nodeName := d.Get("name").(string)
	nodeID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, nodeName)

	node := &types.Node{
		Name:  nodeID,
		Role:  types.NodeRoles[d.Get("role").(string)],
		Image: d.Get("image").(string),
		K3sNodeLabels: map[string]string{
			types.LabelRole: d.Get("role").(string),
		},
		Restart: true,
		Memory:  d.Get("memory").(string),
	}

	if err := client.NodeAddToCluster(ctx, runtimes.SelectedRuntime, node, &types.Cluster{Name: clusterName}, types.NodeCreateOpts{}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(nodeID)

	return resourceNodeRead(ctx, d, meta)
}

func resourceNodeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nodeName := d.Get("name").(string)
	nodeID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, nodeName)

	_, err := client.NodeGet(ctx, runtimes.SelectedRuntime, &types.Node{Name: nodeID})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

/*
func resourceNodeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return diag.Errorf("not implemented")
}
*/

func resourceNodeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nodeName := d.Get("name").(string)
	nodeID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, nodeName)

	if err := client.NodeDelete(ctx, runtimes.SelectedRuntime, &types.Node{Name: nodeID}, types.NodeDeleteOpts{}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
