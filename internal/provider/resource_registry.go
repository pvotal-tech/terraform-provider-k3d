package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/docker/go-connections/nat"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
)

func resourceRegistry() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "k3d-managed registry.",

		CreateContext: resourceRegistryCreate,
		ReadContext:   resourceRegistryRead,
		// UpdateContext: resourceRegistryUpdate,
		DeleteContext: resourceRegistryDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Node name.",
				ForceNew:    true,
				Required:    true,
				Type:        schema.TypeString,
			},
			"image": {
				Description: "Node name.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeString,
				Default:     fmt.Sprintf("%s:%s", types.DefaultRegistryImageRepo, types.DefaultRegistryImageTag),
			},
			"port": {
				Description: "Select which port the registry should be listening on on your machine (localhost).",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeString,
						},
						"host_ip": {
							ForceNew:     true,
							Optional:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.IsIPAddress,
						},
						"host_port": {
							ForceNew:     true,
							Optional:     true,
							Type:         schema.TypeInt,
							ValidateFunc: validation.IsPortNumber,
						},
					},
				},
			},
		},
	}
}

func resourceRegistryCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	registryName := d.Get("name").(string)
	registryID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, registryName)

	registry := &types.Registry{
		ExposureOpts: expandExposureOpts(d.Get("port").([]interface{})),
		Host:         registryID,
		Image:        d.Get("image").(string),
	}

	if _, err := client.RegistryRun(ctx, runtimes.SelectedRuntime, registry); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(registryID)

	return resourceRegistryRead(ctx, d, meta)
}

func resourceRegistryRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	registryName := d.Get("name").(string)
	registryID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, registryName)

	_, err := client.NodeGet(ctx, runtimes.SelectedRuntime, &types.Node{Name: registryID})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

/*
func resourceRegistryUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return diag.Errorf("not implemented")
}
*/

func resourceRegistryDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	registryName := d.Get("name").(string)
	registryID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, registryName)

	if err := client.NodeDelete(ctx, runtimes.SelectedRuntime, &types.Node{Name: registryID}, types.NodeDeleteOpts{}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func expandExposureOpts(l []interface{}) types.ExposureOpts {
	freePort, _ := util.GetFreePort()

	if len(l) == 0 || l[0] == nil {
		return types.ExposureOpts{
			PortMapping: nat.PortMapping{
				Port: types.DefaultRegistryPort,
				Binding: nat.PortBinding{
					HostPort: fmt.Sprintf("%d", freePort),
				},
			},
		}
	}

	v := l[0].(map[string]interface{})

	hostPort := v["host_port"].(int)
	if hostPort == 0 {
		hostPort = freePort
	}

	return types.ExposureOpts{
		PortMapping: nat.PortMapping{
			Port: types.DefaultRegistryPort,
			Binding: nat.PortBinding{
				HostIP:   v["host_ip"].(string),
				HostPort: fmt.Sprintf("%d", hostPort),
			},
		},
	}
}
