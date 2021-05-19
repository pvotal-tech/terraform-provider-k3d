package provider

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/rancher/k3d/v4/cmd/util"
	"github.com/rancher/k3d/v4/pkg/client"
	"github.com/rancher/k3d/v4/pkg/config"
	"github.com/rancher/k3d/v4/pkg/config/v1alpha2"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	"github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/version"
)

func resourceCluster() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Cluster resource in k3d.",

		CreateContext: resourceClusterCreate,
		ReadContext:   resourceClusterRead,
		// UpdateContext: resourceClusterUpdate,
		DeleteContext: resourceClusterDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Cluster name.",
				ForceNew:    true,
				Required:    true,
				Type:        schema.TypeString,
			},
			"agents": {
				Description: "Specify how many agents you want to create.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeInt,
				Default:     0,
			},
			"env": {
				Description: "Add environment variables to nodes.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							ForceNew: true,
							Required: true,
							Type:     schema.TypeString,
						},
						"value": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeString,
						},
						"node_filters": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"image": {
				Description: "Specify k3s image that you want to use for the nodes.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeString,
				Default:     fmt.Sprintf("%s:%s", types.DefaultK3sImageRepo, version.GetK3sVersion(false)),
			},
			"k3d": {
				Description: "k3d runtime settings.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disable_host_ip_injection": {
							Description: "Disable the automatic injection of the Host IP as 'host.k3d.internal' into the containers and CoreDNS.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeBool,
						},
						"disable_image_volume": {
							Description: "Disable the creation of a volume for importing images.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeBool,
						},
						"disable_load_balancer": {
							Description: "Disable the creation of a LoadBalancer in front of the server nodes.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeBool,
						},
					},
				},
			},
			"k3s": {
				Description: "Options passed on to k3s itself.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"extra_agent_args": {
							Description: "Additional args passed to the k3s agent command on agent nodes.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"extra_server_args": {
							Description: "Additional args passed to the k3s server command on server nodes.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"kube_api": {
				ForceNew: true,
				Optional: true,
				Type:     schema.TypeList,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Description: "Important for the `server` setting in the kubeconfig.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeString,
						},
						"host_ip": {
							Description:  "Where the Kubernetes API will be listening on.",
							ForceNew:     true,
							Optional:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.IsIPAddress,
						},
						"host_port": {
							Description:  "Specify the Kubernetes API server port exposed on the LoadBalancer.",
							ForceNew:     true,
							Optional:     true,
							Type:         schema.TypeInt,
							ValidateFunc: validation.IsPortNumber,
						},
					},
				},
			},
			"kubeconfig": {
				Description: "Manage the default kubeconfig",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"update_default_kubeconfig": {
							Description: "Directly update the default kubeconfig with the new cluster's context.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeBool,
							Default:     false,
						},
						"switch_current_context": {
							Description: "Directly switch the default kubeconfig's current-context to the new cluster's context.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeBool,
							Default:     false,
						},
					},
				},
			},
			"kubeconfig_raw": {
				Description: "The full contents of the Kubernetes cluster's kubeconfig file.",
				Computed:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"label": {
				Description: "Add label to node container.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							ForceNew: true,
							Required: true,
							Type:     schema.TypeString,
						},
						"value": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeString,
						},
						"node_filters": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"network": {
				Description: "Join an existing network.",
				Computed:    true,
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeString,
			},
			"port": {
				Description: "Map ports from the node containers to the host.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeString,
						},
						"host_port": {
							ForceNew:     true,
							Optional:     true,
							Type:         schema.TypeInt,
							ValidateFunc: validation.IsPortNumber,
						},
						"container_port": {
							ForceNew:     true,
							Required:     true,
							Type:         schema.TypeInt,
							ValidateFunc: validation.IsPortNumber,
						},
						"protocol": {
							ForceNew:     true,
							Optional:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"TCP", "UDP"}, true),
						},
						"node_filters": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"registries": {
				Description: "Define how registries should be created or used.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"config": {
							Description: "Specify path to an extra registries.yaml file.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeString,
						},
						"create": {
							Description: "Create a k3d-managed registry and connect it to the cluster.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeBool,
						},
						"use": {
							Description: "Connect to one or more k3d-managed registries running locally.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"runtime": {
				Description: "Runtime (Docker) specific options",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"agents_memory": {
							Description: "Memory limit imposed on the agents nodes [From docker].",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeString,
						},
						"gpu_request": {
							Description: "GPU devices to add to the cluster node containers ('all' to pass all GPUs) [From docker].",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeString,
						},
						"servers_memory": {
							Description: "Memory limit imposed on the server nodes [From docker].",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeString,
						},
					},
				},
			},
			"servers": {
				Description: "Specify how many servers you want to create.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeInt,
				Default:     1,
			},
			/*
				"subnet": {
					Description:  "[Experimental: IPAM] Define a subnet for the newly created container network.",
					ForceNew:     true,
					Optional:     true,
					Type:         schema.TypeString,
					ValidateFunc: validation.IsCIDR,
				},
			*/
			"token": {
				Description: "Specify a cluster token. By default, we generate one.",
				Computed:    true,
				ForceNew:    true,
				Optional:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"volume": {
				Description: "Mount volumes into the nodes.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeString,
						},
						"destination": {
							ForceNew: true,
							Required: true,
							Type:     schema.TypeString,
						},
						"node_filters": {
							ForceNew: true,
							Optional: true,
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterName := d.Get("name").(string)

	// TODO: validate all values with GetOk
	simpleConfig := &v1alpha2.SimpleConfig{
		Name:         clusterName,
		Agents:       d.Get("agents").(int),
		ClusterToken: d.Get("token").(string),
		Env:          expandEnvVars(d.Get("env").([]interface{})),
		ExposeAPI:    expandExposureOptions(d.Get("kube_api").([]interface{})),
		Image:        d.Get("image").(string),
		Labels:       expandLabels(d.Get("label").([]interface{})),
		Network:      d.Get("network").(string),
		Ports:        expandPorts(d.Get("port").([]interface{})),
		Servers:      d.Get("servers").(int),
		//Subnet:       d.Get("subnet").(string),
		Volumes: expandVolumes(d.Get("volume").([]interface{})),
	}

	simpleConfig.Options = v1alpha2.SimpleConfigOptions{
		K3dOptions:        expandConfigOptionsK3d(d.Get("k3d").([]interface{})),
		K3sOptions:        expandConfigOptionsK3s(d.Get("k3s").([]interface{})),
		KubeconfigOptions: expandConfigOptionsKubeconfig(d.Get("kubeconfig").([]interface{})),
		Runtime:           expandConfigOptionsRuntime(d.Get("runtime").([]interface{})),
	}

	l := d.Get("registries").([]interface{})
	if len(l) != 0 && l[0] != nil {
		v := l[0].(map[string]interface{})
		simpleConfig.Registries.Config = v["config"].(string)
		simpleConfig.Registries.Create = v["create"].(bool)

		use := make([]string, 0, len(v["use"].([]interface{})))
		for _, i := range v["use"].([]interface{}) {
			use = append(use, i.(string))
		}
		simpleConfig.Registries.Use = use
	}

	// transform simple config to cluster config
	clusterConfig, err := config.TransformSimpleToClusterConfig(ctx, runtimes.SelectedRuntime, *simpleConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	// process cluster config
	clusterConfig, err = config.ProcessClusterConfig(*clusterConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	// validate cluster config
	if err = config.ValidateClusterConfig(ctx, runtimes.SelectedRuntime, *clusterConfig); err != nil {
		return diag.FromErr(err)
	}

	// check if a cluster with that name exists already
	if _, err = client.ClusterGet(ctx, runtimes.SelectedRuntime, &clusterConfig.Cluster); err == nil {
		return diag.Errorf("Failed to create cluster because a cluster with that name already exists")
	}

	// create cluster
	if err = client.ClusterRun(ctx, runtimes.SelectedRuntime, clusterConfig); err != nil {
		// rollback if creation failed
		if deleteErr := client.ClusterDelete(ctx, runtimes.SelectedRuntime, &types.Cluster{Name: clusterName}, types.ClusterDeleteOpts{SkipRegistryCheck: false}); deleteErr != nil {
			return diag.Errorf("Cluster creation FAILED, also FAILED to rollback changes!")
		}
		return diag.FromErr(err)
	}

	// update default kubeconfig
	if clusterConfig.KubeconfigOpts.UpdateDefaultKubeconfig {
		if _, err := client.KubeconfigGetWrite(ctx, runtimes.SelectedRuntime, &clusterConfig.Cluster, "", &client.WriteKubeConfigOptions{UpdateExisting: true, OverwriteExisting: false, UpdateCurrentContext: simpleConfig.Options.KubeconfigOptions.SwitchCurrentContext}); err != nil {
			log.Printf("[WARN] %s", err)
		}
	}

	d.SetId(clusterName)

	return resourceClusterRead(ctx, d, meta)
}

func resourceClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterName := d.Get("name").(string)

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

/*
func resourceClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return diag.Errorf("not implemented")
}
*/

func resourceClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterName := d.Get("name").(string)

	if err := client.ClusterDelete(ctx, runtimes.SelectedRuntime, &types.Cluster{Name: clusterName}, types.ClusterDeleteOpts{SkipRegistryCheck: false}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func expandConfigOptionsK3d(l []interface{}) v1alpha2.SimpleConfigOptionsK3d {
	opts := v1alpha2.SimpleConfigOptionsK3d{
		NoRollback: false,
		Timeout:    0,
		Wait:       true,
	}

	if len(l) == 0 || l[0] == nil {
		return opts
	}

	in := l[0].(map[string]interface{})
	opts.DisableImageVolume = in["disable_image_volume"].(bool)
	opts.DisableLoadbalancer = in["disable_load_balancer"].(bool)
	opts.PrepDisableHostIPInjection = in["disable_host_ip_injection"].(bool)

	return opts
}

func expandConfigOptionsK3s(l []interface{}) v1alpha2.SimpleConfigOptionsK3s {
	if len(l) == 0 || l[0] == nil {
		return v1alpha2.SimpleConfigOptionsK3s{}
	}

	v := l[0].(map[string]interface{})

	extraAgentArgs := make([]string, 0, len(v["extra_agent_args"].([]interface{})))
	for _, i := range v["extra_agent_args"].([]interface{}) {
		extraAgentArgs = append(extraAgentArgs, i.(string))
	}

	extraServerArgs := make([]string, 0, len(v["extra_server_args"].([]interface{})))
	for _, i := range v["extra_server_args"].([]interface{}) {
		extraServerArgs = append(extraServerArgs, i.(string))
	}

	return v1alpha2.SimpleConfigOptionsK3s{
		ExtraAgentArgs:  extraAgentArgs,
		ExtraServerArgs: extraServerArgs,
	}
}

func expandConfigOptionsKubeconfig(l []interface{}) v1alpha2.SimpleConfigOptionsKubeconfig {
	if len(l) == 0 || l[0] == nil {
		return v1alpha2.SimpleConfigOptionsKubeconfig{}
	}

	v := l[0].(map[string]interface{})
	return v1alpha2.SimpleConfigOptionsKubeconfig{
		SwitchCurrentContext:    v["switch_current_context"].(bool),
		UpdateDefaultKubeconfig: v["update_default_kubeconfig"].(bool),
	}
}

func expandConfigOptionsRuntime(l []interface{}) v1alpha2.SimpleConfigOptionsRuntime {
	if len(l) == 0 || l[0] == nil {
		return v1alpha2.SimpleConfigOptionsRuntime{}
	}

	v := l[0].(map[string]interface{})
	return v1alpha2.SimpleConfigOptionsRuntime{
		AgentsMemory:  v["agents_memory"].(string),
		GPURequest:    v["gpu_request"].(string),
		ServersMemory: v["servers_memory"].(string),
	}
}

func expandEnvVars(l []interface{}) []v1alpha2.EnvVarWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	envVars := make([]v1alpha2.EnvVarWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})
		envVars = append(envVars, v1alpha2.EnvVarWithNodeFilters{
			EnvVar:      fmt.Sprintf("%s=%s", v["key"].(string), v["value"].(string)),
			NodeFilters: expandNodeFilters(v["node_filters"].([]interface{})),
		})
	}

	return envVars
}

func expandExposureOptions(l []interface{}) v1alpha2.SimpleExposureOpts {
	freePort, _ := util.GetFreePort()

	if len(l) == 0 || l[0] == nil {
		return v1alpha2.SimpleExposureOpts{
			HostPort: fmt.Sprintf("%d", freePort),
		}
	}

	v := l[0].(map[string]interface{})

	hostPort := v["host_port"].(int)
	if hostPort == 0 {
		hostPort = freePort
	}

	return v1alpha2.SimpleExposureOpts{
		Host:     v["host"].(string),
		HostIP:   v["host_ip"].(string),
		HostPort: fmt.Sprintf("%d", hostPort),
	}
}

func expandLabels(l []interface{}) []v1alpha2.LabelWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	labels := make([]v1alpha2.LabelWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})
		labels = append(labels, v1alpha2.LabelWithNodeFilters{
			Label:       fmt.Sprintf("%s=%s", v["key"].(string), v["value"].(string)),
			NodeFilters: expandNodeFilters(v["node_filters"].([]interface{})),
		})
	}

	return labels
}

func expandNodeFilters(l []interface{}) []string {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	filters := make([]string, 0, len(l))
	for _, i := range l {
		filters = append(filters, i.(string))
	}

	return filters
}

func expandPorts(l []interface{}) []v1alpha2.PortWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	ports := make([]v1alpha2.PortWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})
		ports = append(ports, v1alpha2.PortWithNodeFilters{
			Port:        fmt.Sprintf("%s:%d:%d/%s", v["host"].(string), v["host_port"].(int), v["container_port"].(int), v["protocol"].(string)),
			NodeFilters: expandNodeFilters(v["node_filters"].([]interface{})),
		})
	}

	return ports
}

func expandVolumes(l []interface{}) []v1alpha2.VolumeWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	volumes := make([]v1alpha2.VolumeWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})

		volume := fmt.Sprintf("%s", v["destination"].(string))
		if v["source"].(string) != "" {
			volume = fmt.Sprintf("%s:%s", v["source"].(string), v["destination"].(string))
		}

		volumes = append(volumes, v1alpha2.VolumeWithNodeFilters{
			Volume:      volume,
			NodeFilters: expandNodeFilters(v["node_filters"].([]interface{})),
		})
	}

	return volumes
}
