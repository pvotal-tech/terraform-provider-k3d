package provider

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	types2 "github.com/k3d-io/k3d/v5/pkg/config/types"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/mitchellh/copystructure"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/actions"
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
		UpdateContext: resourceClusterUpdate,
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
				Optional:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Optional: true,
							Type:     schema.TypeString,
						},
						"host_port": {
							Optional:     true,
							Type:         schema.TypeInt,
							ValidateFunc: validation.IsPortNumber,
						},
						"container_port": {
							Required:     true,
							Type:         schema.TypeInt,
							ValidateFunc: validation.IsPortNumber,
						},
						"protocol": {
							Optional:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"TCP", "UDP"}, true),
						},
						"node_filters": {
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

func getSimpleConfig(d *schema.ResourceData) (*v1alpha5.SimpleConfig, error) {
	clusterName := d.Get("name").(string)

	portRaw := d.Get("port")
	var portList []interface{}
	if portRaw != nil {
		var ok bool
		portList, ok = portRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected type for port attribute: %T", portRaw)
		}
	}

	ports, err := expandPorts(portList)
	if err != nil {
		return nil, err
	}

	// TODO: validate all values with GetOk
	simpleConfig := &v1alpha5.SimpleConfig{
		ObjectMeta: types2.ObjectMeta{
			Name: clusterName,
		},
		Agents:       d.Get("agents").(int),
		ClusterToken: d.Get("token").(string),
		Env:          expandEnvVars(d.Get("env").([]interface{})),
		ExposeAPI:    expandExposureOptions(d.Get("kube_api").([]interface{})),
		Image:        d.Get("image").(string),
		Network:      d.Get("network").(string),
		Ports:        ports,
		Servers:      d.Get("servers").(int),
		//Subnet:       d.Get("subnet").(string),
		Volumes: expandVolumes(d.Get("volume").([]interface{})),
		Options: v1alpha5.SimpleConfigOptions{
			Runtime: v1alpha5.SimpleConfigOptionsRuntime{
				Labels: expandLabels(d.Get("label").([]interface{})),
			},
		},
	}

	simpleConfig.Options = v1alpha5.SimpleConfigOptions{
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
			simpleConfig.Registries.Create = &v1alpha5.SimpleConfigRegistryCreateConfig{
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

	return simpleConfig, nil
}

func getClusterConfig(ctx context.Context, simpleConfig v1alpha5.SimpleConfig) (*v1alpha5.ClusterConfig, error) {
	// transform simple config to cluster config
	configFileName := "" // new embedded and external config files is not supported: https://github.com/k3d-io/k3d/pull/1417
	clusterConfig, err := config.TransformSimpleToClusterConfig(ctx, runtimes.SelectedRuntime, simpleConfig, configFileName)
	if err != nil {
		return nil, err
	}

	// process cluster config
	clusterConfig, err = config.ProcessClusterConfig(*clusterConfig)
	if err != nil {
		return nil, err
	}

	// validate cluster config
	if err = config.ValidateClusterConfig(ctx, runtimes.SelectedRuntime, *clusterConfig); err != nil {
		return nil, err
	}

	return clusterConfig, nil
}

func resourceClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterName := d.Get("name").(string)

	simpleConfig, err := getSimpleConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterConfig, err := getClusterConfig(ctx, *simpleConfig)
	if err != nil {
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

func resourceClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChange("port") {
		return resourceClusterRead(ctx, d, meta)
	}

	clusterName := d.Get("name").(string)

	cluster, err := client.ClusterGet(ctx, runtimes.SelectedRuntime, &types.Cluster{Name: clusterName})
	if err != nil {
		return diag.FromErr(err)
	}

	oldRaw, newRaw := d.GetChange("port")
	oldPorts, err := expandPortsFromRaw(oldRaw)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to parse previous port configuration: %w", err))
	}

	newPorts, err := expandPortsFromRaw(newRaw)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to parse desired port configuration: %w", err))
	}

	oldProjection, err := buildClusterPortProjection(ctx, cluster, oldPorts)
	if err != nil {
		return diag.FromErr(err)
	}

	newProjection, err := buildClusterPortProjection(ctx, cluster, newPorts)
	if err != nil {
		return diag.FromErr(err)
	}

	plan := determinePortUpdatePlan(oldProjection, newProjection)

	if plan.loadBalancer || len(plan.nodeNames) > 0 {
		if err := applyPortUpdatePlan(ctx, cluster, newProjection, plan); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceClusterRead(ctx, d, meta)
}

func resourceClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterName := d.Get("name").(string)

	if err := client.ClusterDelete(ctx, runtimes.SelectedRuntime, &types.Cluster{Name: clusterName}, types.ClusterDeleteOpts{SkipRegistryCheck: false}); err != nil {
		return diag.FromErr(err)
	}

	simpleConfig, err := getSimpleConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterConfig, err := getClusterConfig(ctx, *simpleConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	// clean up default kubeconfig
	if err := client.KubeconfigRemoveClusterFromDefaultConfig(ctx, &clusterConfig.Cluster); err != nil {
		log.Printf("[WARN] Failed to remove cluster details from default kubeconfig")
		log.Printf("[WARN] %s", err)
	}

	return nil
}

type portUpdatePlan struct {
	loadBalancer bool
	nodeNames    []string
}

func expandPortsFromRaw(raw interface{}) ([]v1alpha5.PortWithNodeFilters, error) {
	if raw == nil {
		return nil, nil
	}

	list, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type for port diff: %T", raw)
	}

	if len(list) == 0 {
		return nil, nil
	}

	return expandPorts(list)
}

func buildClusterPortProjection(ctx context.Context, reference *types.Cluster, ports []v1alpha5.PortWithNodeFilters) (*types.Cluster, error) {
	cloneValue, err := copystructure.Copy(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to copy cluster definition: %w", err)
	}

	clone, ok := cloneValue.(*types.Cluster)
	if !ok {
		return nil, fmt.Errorf("unexpected cluster copy type %T", cloneValue)
	}

	resetClusterPortState(clone, reference)

	if len(ports) == 0 {
		return clone, nil
	}

	if err := client.TransformPorts(ctx, runtimes.SelectedRuntime, clone, ports); err != nil {
		return nil, fmt.Errorf("failed to apply port configuration: %w", err)
	}

	return clone, nil
}

func resetClusterPortState(target *types.Cluster, reference *types.Cluster) {
	if target == nil {
		return
	}

	servers := collectServerNames(target)

	for _, node := range target.Nodes {
		if node == nil {
			continue
		}
		node.Ports = nat.PortMap{}
	}

	if target.ServerLoadBalancer == nil || target.ServerLoadBalancer.Node == nil {
		return
	}

	bindings := []nat.PortBinding{}
	if reference != nil && reference.ServerLoadBalancer != nil && reference.ServerLoadBalancer.Node != nil {
		if existing, ok := reference.ServerLoadBalancer.Node.Ports[types.DefaultAPIPort]; ok {
			bindings = copyPortBindings(existing)
		}
	}

	if len(bindings) == 0 && reference != nil && reference.KubeAPI != nil {
		bindings = append(bindings, reference.KubeAPI.Binding)
	}
	if len(bindings) == 0 && target.KubeAPI != nil {
		bindings = append(bindings, target.KubeAPI.Binding)
	}

	target.ServerLoadBalancer.Node.Ports = nat.PortMap{}
	if len(bindings) > 0 {
		target.ServerLoadBalancer.Node.Ports[types.DefaultAPIPort] = bindings
	}

	settings := types.LoadBalancerSettings{
		WorkerConnections: types.DefaultLoadbalancerWorkerConnections,
	}
	if reference != nil && reference.ServerLoadBalancer != nil && reference.ServerLoadBalancer.Config != nil {
		settings = reference.ServerLoadBalancer.Config.Settings
	}

	portKey := fmt.Sprintf("%s.tcp", types.DefaultAPIPort)
	target.ServerLoadBalancer.Config = &types.LoadbalancerConfig{
		Ports: map[string][]string{
			portKey: append([]string(nil), servers...),
		},
		Settings: settings,
	}
}

func collectServerNames(cluster *types.Cluster) []string {
	if cluster == nil {
		return nil
	}

	servers := make([]string, 0)
	for _, node := range cluster.Nodes {
		if node != nil && node.Role == types.ServerRole {
			servers = append(servers, node.Name)
		}
	}

	return servers
}

func copyPortMap(src nat.PortMap) nat.PortMap {
	if len(src) == 0 {
		return nat.PortMap{}
	}

	dst := make(nat.PortMap, len(src))
	for port, bindings := range src {
		dst[port] = copyPortBindings(bindings)
	}

	return dst
}

func copyPortBindings(src []nat.PortBinding) []nat.PortBinding {
	if len(src) == 0 {
		return nil
	}

	dst := make([]nat.PortBinding, len(src))
	copy(dst, src)
	return dst
}

func portMapEqual(a, b nat.PortMap) bool {
	if len(a) != len(b) {
		return false
	}

	for port, bindingsA := range a {
		bindingsB, ok := b[port]
		if !ok {
			return false
		}
		if !portBindingsEqual(bindingsA, bindingsB) {
			return false
		}
	}

	return true
}

func portBindingsEqual(a, b []nat.PortBinding) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}

	copyA := make([]nat.PortBinding, len(a))
	copy(copyA, a)
	copyB := make([]nat.PortBinding, len(b))
	copy(copyB, b)

	sort.Slice(copyA, func(i, j int) bool {
		return copyA[i].HostIP+":"+copyA[i].HostPort < copyA[j].HostIP+":"+copyA[j].HostPort
	})
	sort.Slice(copyB, func(i, j int) bool {
		return copyB[i].HostIP+":"+copyB[i].HostPort < copyB[j].HostIP+":"+copyB[j].HostPort
	})

	for i := range copyA {
		if copyA[i] != copyB[i] {
			return false
		}
	}

	return true
}

func determinePortUpdatePlan(oldProjection, newProjection *types.Cluster) portUpdatePlan {
	plan := portUpdatePlan{}

	if !portMapEqual(getLoadbalancerPorts(oldProjection), getLoadbalancerPorts(newProjection)) {
		plan.loadBalancer = true
	}

	oldNodes := mapNodesByName(oldProjection)
	newNodes := mapNodesByName(newProjection)

	for name, newNode := range newNodes {
		if newNode == nil || newNode.Role == types.LoadBalancerRole {
			continue
		}

		oldNode := oldNodes[name]
		var oldPorts nat.PortMap
		if oldNode != nil {
			oldPorts = oldNode.Ports
		}
		if !portMapEqual(oldPorts, newNode.Ports) {
			plan.nodeNames = append(plan.nodeNames, name)
		}
	}

	if len(plan.nodeNames) > 1 {
		sort.Strings(plan.nodeNames)
	}

	return plan
}

func getLoadbalancerPorts(cluster *types.Cluster) nat.PortMap {
	if cluster == nil || cluster.ServerLoadBalancer == nil || cluster.ServerLoadBalancer.Node == nil {
		return nat.PortMap{}
	}
	return cluster.ServerLoadBalancer.Node.Ports
}

func mapNodesByName(cluster *types.Cluster) map[string]*types.Node {
	result := make(map[string]*types.Node)
	if cluster == nil {
		return result
	}

	for _, node := range cluster.Nodes {
		if node != nil {
			result[node.Name] = node
		}
	}

	return result
}

func applyPortUpdatePlan(ctx context.Context, actual *types.Cluster, desired *types.Cluster, plan portUpdatePlan) error {
	if plan.loadBalancer {
		if err := replaceLoadBalancer(ctx, actual, desired); err != nil {
			return err
		}
	}

	for _, name := range plan.nodeNames {
		if err := replaceClusterNode(ctx, actual, desired, name); err != nil {
			return err
		}
	}

	return nil
}

func replaceLoadBalancer(ctx context.Context, actual *types.Cluster, desired *types.Cluster) error {
	if actual.ServerLoadBalancer == nil || desired.ServerLoadBalancer == nil {
		return fmt.Errorf("cluster does not have a load balancer")
	}

	replacement, err := client.CopyNode(ctx, actual.ServerLoadBalancer.Node, client.CopyNodeOpts{})
	if err != nil {
		return fmt.Errorf("failed to copy load balancer node: %w", err)
	}

	replacement.Ports = copyPortMap(desired.ServerLoadBalancer.Node.Ports)
	replacement.HookActions = filterLoadbalancerConfigHooks(replacement.HookActions)

	lbConfig, err := client.LoadbalancerGenerateConfig(desired)
	if err != nil {
		return fmt.Errorf("failed to generate load balancer config: %w", err)
	}

	configyaml, err := yaml.Marshal(lbConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal load balancer config: %w", err)
	}

	replacement.HookActions = append(replacement.HookActions, types.NodeHook{
		Stage: types.LifecycleStagePreStart,
		Action: actions.WriteFileAction{
			Runtime:     runtimes.SelectedRuntime,
			Content:     configyaml,
			Dest:        types.DefaultLoadbalancerConfigPath,
			Mode:        0o744,
			Description: "Write Loadbalancer Configuration",
		},
	})

	if err := client.NodeReplace(ctx, runtimes.SelectedRuntime, actual.ServerLoadBalancer.Node, replacement); err != nil {
		return fmt.Errorf("failed to replace load balancer node: %w", err)
	}

	actual.ServerLoadBalancer.Node = replacement
	actual.ServerLoadBalancer.Config = &lbConfig

	return nil
}

func filterLoadbalancerConfigHooks(hooks []types.NodeHook) []types.NodeHook {
	if len(hooks) == 0 {
		return hooks
	}

	filtered := make([]types.NodeHook, 0, len(hooks))
	for _, hook := range hooks {
		if hook.Action == nil {
			filtered = append(filtered, hook)
			continue
		}

		skip := false
		switch act := hook.Action.(type) {
		case actions.WriteFileAction:
			if act.Dest == types.DefaultLoadbalancerConfigPath {
				skip = true
			}
		case *actions.WriteFileAction:
			if act != nil && act.Dest == types.DefaultLoadbalancerConfigPath {
				skip = true
			}
		}

		if skip {
			continue
		}

		filtered = append(filtered, hook)
	}

	return filtered
}

func replaceClusterNode(ctx context.Context, actual *types.Cluster, desired *types.Cluster, name string) error {
	current := findNodeByName(actual, name)
	desiredNode := findNodeByName(desired, name)
	if current == nil || desiredNode == nil {
		return fmt.Errorf("node %s not found", name)
	}

	replacement, err := client.CopyNode(ctx, current, client.CopyNodeOpts{})
	if err != nil {
		return fmt.Errorf("failed to copy node %s: %w", name, err)
	}

	replacement.Ports = copyPortMap(desiredNode.Ports)

	if err := client.NodeReplace(ctx, runtimes.SelectedRuntime, current, replacement); err != nil {
		return fmt.Errorf("failed to replace node %s: %w", name, err)
	}

	for i, node := range actual.Nodes {
		if node != nil && node.Name == name {
			actual.Nodes[i] = replacement
			break
		}
	}

	return nil
}

func findNodeByName(cluster *types.Cluster, name string) *types.Node {
	if cluster == nil {
		return nil
	}

	for _, node := range cluster.Nodes {
		if node != nil && node.Name == name {
			return node
		}
	}

	return nil
}

func expandConfigOptionsK3d(l []interface{}) v1alpha5.SimpleConfigOptionsK3d {
	opts := v1alpha5.SimpleConfigOptionsK3d{
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

func expandConfigOptionsK3s(l []interface{}) v1alpha5.SimpleConfigOptionsK3s {
	if len(l) == 0 || l[0] == nil {
		return v1alpha5.SimpleConfigOptionsK3s{}
	}

	v := l[0].(map[string]interface{})

	extraArgs := make([]v1alpha5.K3sArgWithNodeFilters, 0)
	for _, i := range v["extra_args"].([]interface{}) {

		extraArgs = append(extraArgs, v1alpha5.K3sArgWithNodeFilters{
			Arg:         i.(map[string]interface{})["arg"].(string),
			NodeFilters: expandNodeFilters(i.(map[string]interface{})["node_filters"].([]interface{})),
		})
	}

	return v1alpha5.SimpleConfigOptionsK3s{
		ExtraArgs: extraArgs,
	}
}

func expandConfigOptionsKubeconfig(l []interface{}) v1alpha5.SimpleConfigOptionsKubeconfig {
	if len(l) == 0 || l[0] == nil {
		return v1alpha5.SimpleConfigOptionsKubeconfig{}
	}

	v := l[0].(map[string]interface{})
	return v1alpha5.SimpleConfigOptionsKubeconfig{
		SwitchCurrentContext:    v["switch_current_context"].(bool),
		UpdateDefaultKubeconfig: v["update_default_kubeconfig"].(bool),
	}
}

func expandConfigOptionsRuntime(l []interface{}) v1alpha5.SimpleConfigOptionsRuntime {
	if len(l) == 0 || l[0] == nil {
		return v1alpha5.SimpleConfigOptionsRuntime{}
	}

	v := l[0].(map[string]interface{})
	return v1alpha5.SimpleConfigOptionsRuntime{
		AgentsMemory:  v["agents_memory"].(string),
		GPURequest:    v["gpu_request"].(string),
		ServersMemory: v["servers_memory"].(string),
	}
}

func expandEnvVars(l []interface{}) []v1alpha5.EnvVarWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	envVars := make([]v1alpha5.EnvVarWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})
		envVars = append(envVars, v1alpha5.EnvVarWithNodeFilters{
			EnvVar:      fmt.Sprintf("%s=%s", v["key"].(string), v["value"].(string)),
			NodeFilters: expandNodeFilters(v["node_filters"].([]interface{})),
		})
	}

	return envVars
}

func expandExposureOptions(l []interface{}) v1alpha5.SimpleExposureOpts {
	freePort, _ := util.GetFreePort()

	if len(l) == 0 || l[0] == nil {
		return v1alpha5.SimpleExposureOpts{
			HostPort: fmt.Sprintf("%d", freePort),
		}
	}

	v := l[0].(map[string]interface{})

	hostPort := v["host_port"].(int)
	if hostPort == 0 {
		hostPort = freePort
	}

	return v1alpha5.SimpleExposureOpts{
		Host:     v["host"].(string),
		HostIP:   v["host_ip"].(string),
		HostPort: fmt.Sprintf("%d", hostPort),
	}
}

func expandLabels(l []interface{}) []v1alpha5.LabelWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	labels := make([]v1alpha5.LabelWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})
		labels = append(labels, v1alpha5.LabelWithNodeFilters{
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

func expandPorts(l []interface{}) ([]v1alpha5.PortWithNodeFilters, error) {
	if len(l) == 0 {
		return nil, nil
	}
	if len(l) == 1 && l[0] == nil {
		return nil, nil
	}

	ports := make([]v1alpha5.PortWithNodeFilters, 0, len(l))
	for idx, item := range l {
		if item == nil {
			return nil, fmt.Errorf("port[%d]: expected object, got nil", idx)
		}

		entry, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("port[%d]: expected object, got %T", idx, item)
		}

		host := ""
		if value, exists := entry["host"]; exists {
			if value == nil {
				host = ""
			} else {
				str, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("port[%d].host: expected string, got %T", idx, value)
				}
				host = str
			}
		}

		hostPort := 0
		if value, exists := entry["host_port"]; exists {
			if value != nil {
				num, ok := value.(int)
				if !ok {
					return nil, fmt.Errorf("port[%d].host_port: expected number, got %T", idx, value)
				}
				hostPort = num
			}
		}

		containerValue, exists := entry["container_port"]
		if !exists || containerValue == nil {
			return nil, fmt.Errorf("port[%d].container_port: missing value", idx)
		}

		containerPort, ok := containerValue.(int)
		if !ok {
			return nil, fmt.Errorf("port[%d].container_port: expected number, got %T", idx, containerValue)
		}

		protocol := ""
		if value, exists := entry["protocol"]; exists {
			if value == nil {
				protocol = ""
			} else {
				str, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("port[%d].protocol: expected string, got %T", idx, value)
				}
				protocol = str
			}
		}

		var filtersList []interface{}
		if value, exists := entry["node_filters"]; exists {
			if value != nil {
				list, ok := value.([]interface{})
				if !ok {
					return nil, fmt.Errorf("port[%d].node_filters: expected list, got %T", idx, value)
				}
				filtersList = list
			}
		}

		ports = append(ports, v1alpha5.PortWithNodeFilters{
			Port:        fmt.Sprintf("%s:%d:%d/%s", host, hostPort, containerPort, protocol),
			NodeFilters: expandNodeFilters(filtersList),
		})
	}

	return ports, nil
}

func expandVolumes(l []interface{}) []v1alpha5.VolumeWithNodeFilters {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	volumes := make([]v1alpha5.VolumeWithNodeFilters, 0, len(l))
	for _, i := range l {
		v := i.(map[string]interface{})

		volume := fmt.Sprintf("%s", v["destination"].(string))
		if v["source"].(string) != "" {
			volume = fmt.Sprintf("%s:%s", v["source"].(string), v["destination"].(string))
		}

		volumes = append(volumes, v1alpha5.VolumeWithNodeFilters{
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
