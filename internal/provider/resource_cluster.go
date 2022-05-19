package provider

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	types2 "github.com/k3d-io/k3d/v5/pkg/config/types"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha4"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/config"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/version"
)

func resourceCluster() *schema.Resource {

	k3sVersion, err := version.GetK3sVersion("stable")
	if err != nil {
		panic(err)
	}

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
			"credentials": {
				Description: "Cluster credentials.",
				Computed:    true,
				Sensitive:   true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client_certificate": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"client_key": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"cluster_ca_certificate": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"host": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"raw": {
							Computed: true,
							Type:     schema.TypeString,
						},
					},
				},
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
				Default:     fmt.Sprintf("%s:%s", types.DefaultK3sImageRepo, k3sVersion),
			},
			"k3d": {
				Description: "k3d runtime settings.",
				ForceNew:    true,
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
						"extra_args": {
							Description: "Additional args passed to the k3s command.",
							ForceNew:    true,
							Optional:    true,
							Type:        schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"arg": {
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
							Type:        schema.TypeList,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Description: "Name of the registry to create.",
										ForceNew:    true,
										Optional:    true,
										Type:        schema.TypeString,
									},
									"host": {
										Description: "Hostname to link to the created registry.",
										ForceNew:    true,
										Optional:    true,
										Type:        schema.TypeString,
									},
									"image": {
										Description: "Docker image of the registry.",
										ForceNew:    true,
										Optional:    true,
										Type:        schema.TypeString,
									},
									"host_port": {
										Description: "Host port exposed to access the registry.",
										ForceNew:    true,
										Optional:    true,
										Type:        schema.TypeString,
									},
								},
							},
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
	simpleConfig := &v1alpha4.SimpleConfig{
		ObjectMeta: types2.ObjectMeta{
			Name: clusterName,
		},
		Agents:       d.Get("agents").(int),
		ClusterToken: d.Get("token").(string),
		Env:          expandEnvVars(d.Get("env").([]interface{})),
		ExposeAPI:    expandExposureOptions(d.Get("kube_api").([]interface{})),
		Image:        d.Get("image").(string),
		Network:      d.Get("network").(string),
		Ports:        expandPorts(d.Get("port").([]interface{})),
		Servers:      d.Get("servers").(int),
		//Subnet:       d.Get("subnet").(string),
		Volumes: expandVolumes(d.Get("volume").([]interface{})),
		Options: v1alpha4.SimpleConfigOptions{
			Runtime: v1alpha4.SimpleConfigOptionsRuntime{
				Labels: expandLabels(d.Get("label").([]interface{})),
			},
		},
	}

	simpleConfig.Options = v1alpha4.SimpleConfigOptions{
		K3dOptions:        expandConfigOptionsK3d(d.Get("k3d").([]interface{})),
		K3sOptions:        expandConfigOptionsK3s(d.Get("k3s").([]interface{})),
		KubeconfigOptions: expandConfigOptionsKubeconfig(d.Get("kubeconfig").([]interface{})),
		Runtime:           expandConfigOptionsRuntime(d.Get("runtime").([]interface{})),
	}

	l := d.Get("registries").([]interface{})
	if len(l) != 0 && l[0] != nil {
		v := l[0].(map[string]interface{})
		simpleConfig.Registries.Config = v["config"].(string)
		registryToCreate := v["create"].([]interface{})
		if len(registryToCreate) == 1 {
			rtc := registryToCreate[0].(map[string]interface{})
			simpleConfig.Registries.Create = &v1alpha4.SimpleConfigRegistryCreateConfig{
				Name:     rtc["name"].(string),
				Host:     rtc["host"].(string),
				Image:    rtc["image"].(string),
				HostPort: rtc["host_port"].(string),
			}
		}

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
		if err == nil {
			if err := d.Set("credentials", flattenCredentials(clusterName, k)); err != nil {
				return diag.FromErr(err)
			}
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

func expandConfigOptionsK3d(l []interface{}) v1alpha4.SimpleConfigOptionsK3d {
	opts := v1alpha4.SimpleConfigOptionsK3d{
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

	return opts
}

func expandConfigOptionsK3s(l []interface{}) v1alpha4.SimpleConfigOptionsK3s {
	if len(l) == 0 || l[0] == nil {
		return v1alpha4.SimpleConfigOptionsK3s{}
	}

	v := l[0].(map[string]interface{})

	extraArgs := make([]v1alpha4.K3sArgWithNodeFilters, 0)
	for _, i := range v["extra_args"].([]interface{}) {

		extraArgs = append(extraArgs, v1alpha4.K3sArgWithNodeFilters{
			Arg:         i.(map[string]interface{})["arg"].(string),
			NodeFilters: expandNodeFilters(i.(map[string]interface{})["node_filters"].([]interface{})),
		})
	}

	return v1alpha4.SimpleConfigOptionsK3s{
		ExtraArgs: extraArgs,
	}
}

func expandConfigOptionsKubeconfig(l []interface{}) v1alpha4.SimpleConfigOptionsKubeconfig {
	if len(l) == 0 || l[0] == nil {
		return v1alpha4.SimpleConfigOptionsKubeconfig{}
	}

	v := l[0].(map[string]interface{})
	return v1alpha4.SimpleConfigOptionsKubeconfig{
		SwitchCurrentContext:    v["switch_current_context"].(bool),
		UpdateDefaultKubeconfig: v["update_default_kubeconfig"].(bool),
	}
}

func expandConfigOptionsRuntime(l []interface{}) v1alpha4.SimpleConfigOptionsRuntime {
	if len(l) == 0 || l[0] == nil {
		return v1alpha4.SimpleConfigOptionsRuntime{}
	}

	v := l[0].(map[string]interface{})
	return v1alpha4.SimpleConfigOptionsRuntime{
		AgentsMemory:  v["agents_memory"].(string),
		GPURequest:    v["gpu_request"].(string),
		ServersMemory: v["servers_memory"].(string),
	}
}

func expandEnvVars(l []interface{}) []v1alpha4.EnvVarWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	envVars := make([]v1alpha4.EnvVarWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})
		envVars = append(envVars, v1alpha4.EnvVarWithNodeFilters{
			EnvVar:      fmt.Sprintf("%s=%s", v["key"].(string), v["value"].(string)),
			NodeFilters: expandNodeFilters(v["node_filters"].([]interface{})),
		})
	}

	return envVars
}

func expandExposureOptions(l []interface{}) v1alpha4.SimpleExposureOpts {
	freePort, _ := util.GetFreePort()

	if len(l) == 0 || l[0] == nil {
		return v1alpha4.SimpleExposureOpts{
			HostPort: fmt.Sprintf("%d", freePort),
		}
	}

	v := l[0].(map[string]interface{})

	hostPort := v["host_port"].(int)
	if hostPort == 0 {
		hostPort = freePort
	}

	return v1alpha4.SimpleExposureOpts{
		Host:     v["host"].(string),
		HostIP:   v["host_ip"].(string),
		HostPort: fmt.Sprintf("%d", hostPort),
	}
}

func expandLabels(l []interface{}) []v1alpha4.LabelWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	labels := make([]v1alpha4.LabelWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})
		labels = append(labels, v1alpha4.LabelWithNodeFilters{
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

func expandPorts(l []interface{}) []v1alpha4.PortWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	ports := make([]v1alpha4.PortWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})
		ports = append(ports, v1alpha4.PortWithNodeFilters{
			Port:        fmt.Sprintf("%s:%d:%d/%s", v["host"].(string), v["host_port"].(int), v["container_port"].(int), v["protocol"].(string)),
			NodeFilters: expandNodeFilters(v["node_filters"].([]interface{})),
		})
	}

	return ports
}

func expandVolumes(l []interface{}) []v1alpha4.VolumeWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	volumes := make([]v1alpha4.VolumeWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})

		volume := fmt.Sprintf("%s", v["destination"].(string))
		if v["source"].(string) != "" {
			volume = fmt.Sprintf("%s:%s", v["source"].(string), v["destination"].(string))
		}

		volumes = append(volumes, v1alpha4.VolumeWithNodeFilters{
			Volume:      volume,
			NodeFilters: expandNodeFilters(v["node_filters"].([]interface{})),
		})
	}

	return volumes
}

func flattenCredentials(clusterName string, config *clientcmdapi.Config) []interface{} {
	clusterID := fmt.Sprintf("%s-%s", types.DefaultObjectNamePrefix, clusterName)
	authInfoName := fmt.Sprintf("admin@%s-%s", types.DefaultObjectNamePrefix, clusterName)

	raw, _ := clientcmd.Write(*config)

	creds := map[string]interface{}{
		"client_certificate":     string(config.AuthInfos[authInfoName].ClientCertificateData),
		"client_key":             string(config.AuthInfos[authInfoName].ClientKeyData),
		"cluster_ca_certificate": string(config.Clusters[clusterID].CertificateAuthorityData),
		"host":                   config.Clusters[clusterID].Server,
		"raw":                    string(raw),
	}

	return []interface{}{creds}
}
