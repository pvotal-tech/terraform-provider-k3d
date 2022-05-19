package provider

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
)

func dataSourceCluster() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Cluster data source in k3d.",

		ReadContext: dataSourceClusterRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Cluster name.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"kubeconfig_raw": {
				Description: "The full contents of the Kubernetes cluster's kubeconfig file.",
				Computed:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"network": {
				Description: "Join an existing network.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"token": {
				Description: "Specify a cluster token. By default, we generate one.",
				Computed:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
		},
	}
}

func dataSourceClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterName := d.Get("name").(string)
	d.SetId(clusterName)

	cluster, err := client.ClusterGet(ctx, runtimes.SelectedRuntime, &types.Cluster{Name: clusterName})
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("network", cluster.Network.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("token", cluster.Token); err != nil {
		return diag.FromErr(err)
	}

	k, err := client.KubeconfigGet(ctx, runtimes.SelectedRuntime, cluster)
	if err == nil {
		r, err := clientcmd.Write(*k)
		if err == nil {
			d.Set("kubeconfig_raw", fmt.Sprintf("%s", r))
		} else {
			log.Printf("[WARN] %s", err)
		}
	} else {
		log.Printf("[WARN] %s", err)
	}

	return nil
}
