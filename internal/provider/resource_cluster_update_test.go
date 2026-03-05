package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/k3d-io/k3d/v5/pkg/types"
)

func TestDeterminePortUpdatePlan_LoadBalancerChange(t *testing.T) {
	cluster := testClusterFixture()
	ctx := context.Background()

	oldPorts := []v1alpha5.PortWithNodeFilters{
		{Port: "8080:80/tcp", NodeFilters: []string{"loadbalancer"}},
	}
	newPorts := []v1alpha5.PortWithNodeFilters{
		{Port: "9090:80/tcp", NodeFilters: []string{"loadbalancer"}},
	}

	oldProjection, err := buildClusterPortProjection(ctx, cluster, oldPorts)
	if err != nil {
		t.Fatalf("failed to build old projection: %v", err)
	}

	newProjection, err := buildClusterPortProjection(ctx, cluster, newPorts)
	if err != nil {
		t.Fatalf("failed to build new projection: %v", err)
	}

	plan := determinePortUpdatePlan(oldProjection, newProjection)
	if !plan.loadBalancer {
		t.Fatalf("expected load balancer update, got %+v", plan)
	}
	if len(plan.nodeNames) != 0 {
		t.Fatalf("expected no node replacements, got %v", plan.nodeNames)
	}
}

func TestDeterminePortUpdatePlan_DirectNodeChange(t *testing.T) {
	cluster := testClusterFixture()
	ctx := context.Background()

	oldPorts := []v1alpha5.PortWithNodeFilters{
		{Port: "30000:30000/tcp", NodeFilters: []string{"servers:0:direct"}},
	}
	newPorts := []v1alpha5.PortWithNodeFilters{
		{Port: "31000:30000/tcp", NodeFilters: []string{"servers:0:direct"}},
	}

	oldProjection, err := buildClusterPortProjection(ctx, cluster, oldPorts)
	if err != nil {
		t.Fatalf("failed to build old projection: %v", err)
	}

	newProjection, err := buildClusterPortProjection(ctx, cluster, newPorts)
	if err != nil {
		t.Fatalf("failed to build new projection: %v", err)
	}

	plan := determinePortUpdatePlan(oldProjection, newProjection)
	if plan.loadBalancer {
		t.Fatalf("did not expect load balancer update for direct mapping")
	}
	if len(plan.nodeNames) != 1 || plan.nodeNames[0] != "k3d-test-server-0" {
		t.Fatalf("expected server node update, got %v", plan.nodeNames)
	}
}

func TestDeterminePortUpdatePlan_NoChanges(t *testing.T) {
	cluster := testClusterFixture()
	ctx := context.Background()

	ports := []v1alpha5.PortWithNodeFilters{
		{Port: "8080:80/tcp", NodeFilters: []string{"loadbalancer"}},
	}

	oldProjection, err := buildClusterPortProjection(ctx, cluster, ports)
	if err != nil {
		t.Fatalf("failed to build old projection: %v", err)
	}

	newProjection, err := buildClusterPortProjection(ctx, cluster, ports)
	if err != nil {
		t.Fatalf("failed to build new projection: %v", err)
	}

	plan := determinePortUpdatePlan(oldProjection, newProjection)
	if plan.loadBalancer {
		t.Fatalf("did not expect load balancer update")
	}
	if len(plan.nodeNames) != 0 {
		t.Fatalf("did not expect node updates, got %v", plan.nodeNames)
	}
}

func TestBuildClusterPortProjectionRetainsKubeAPI(t *testing.T) {
	cluster := testClusterFixture()
	ctx := context.Background()

	ports := []v1alpha5.PortWithNodeFilters{
		{Port: "8080:80/tcp", NodeFilters: []string{"loadbalancer"}},
	}

	projection, err := buildClusterPortProjection(ctx, cluster, ports)
	if err != nil {
		t.Fatalf("buildClusterPortProjection failed: %v", err)
	}

	bindings, ok := projection.ServerLoadBalancer.Node.Ports[types.DefaultAPIPort]
	if !ok || len(bindings) == 0 {
		t.Fatalf("kube API port %s missing from load balancer ports", types.DefaultAPIPort)
	}

	expectedBinding := cluster.KubeAPI.Binding
	if bindings[0].HostPort != expectedBinding.HostPort {
		t.Fatalf("expected kube API host port %s, got %s", expectedBinding.HostPort, bindings[0].HostPort)
	}
	if bindings[0].HostIP != expectedBinding.HostIP {
		t.Fatalf("expected kube API host IP %s, got %s", expectedBinding.HostIP, bindings[0].HostIP)
	}

	portKey := fmt.Sprintf("%s.tcp", types.DefaultAPIPort)
	targets, ok := projection.ServerLoadBalancer.Config.Ports[portKey]
	if !ok || len(targets) == 0 {
		t.Fatalf("kube API port %s missing from load balancer config", portKey)
	}
	if targets[0] != "k3d-test-server-0" {
		t.Fatalf("expected server target k3d-test-server-0, got %v", targets)
	}
}

func TestEnsureKubeAPIPublishedRestoresBinding(t *testing.T) {
	reference := testClusterFixture()
	target := testClusterFixture()

	delete(target.ServerLoadBalancer.Node.Ports, types.DefaultAPIPort)
	target.ServerLoadBalancer.Config = &types.LoadbalancerConfig{
		Ports:    map[string][]string{},
		Settings: types.LoadBalancerSettings{},
	}

	ensureKubeAPIPublished(target, reference)

	bindings, ok := target.ServerLoadBalancer.Node.Ports[types.DefaultAPIPort]
	if !ok || len(bindings) == 0 {
		t.Fatalf("kube API port %s not restored", types.DefaultAPIPort)
	}

	expected := reference.KubeAPI.Binding
	if bindings[0] != expected {
		t.Fatalf("expected binding %+v, got %+v", expected, bindings[0])
	}

	portKey := fmt.Sprintf("%s.tcp", types.DefaultAPIPort)
	targets, ok := target.ServerLoadBalancer.Config.Ports[portKey]
	if !ok || len(targets) == 0 {
		t.Fatalf("load balancer config missing kube API targets for %s", portKey)
	}

	if targets[0] != "k3d-test-server-0" {
		t.Fatalf("expected kube API target k3d-test-server-0, got %v", targets)
	}
}

func TestReplaceLoadBalancerRequiresKubeAPIPort(t *testing.T) {
	ctx := context.Background()
	actual := testClusterFixture()
	desired := testClusterFixture()
	delete(desired.ServerLoadBalancer.Node.Ports, types.DefaultAPIPort)

	err := replaceLoadBalancer(ctx, actual, desired)
	if err == nil {
		t.Fatal("expected error when kube API port missing")
	}

	if !strings.Contains(err.Error(), types.DefaultAPIPort) {
		t.Fatalf("expected error to mention kube API port, got %v", err)
	}
}

func TestExpandPortsFromRawRejectsUnexpectedShape(t *testing.T) {
	if _, err := expandPortsFromRaw(map[string]interface{}{}); err == nil {
		t.Fatal("expected error for non-slice raw value")
	}
}

func TestExpandPortsRejectsMalformedEntry(t *testing.T) {
	_, err := expandPorts([]interface{}{
		map[string]interface{}{
			"host": "127.0.0.1",
		},
	})
	if err == nil {
		t.Fatal("expected error for missing container_port")
	}
}

func testClusterFixture() *types.Cluster {
	apiPort := nat.Port(fmt.Sprintf("%s/tcp", types.DefaultAPIPort))
	apiBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: types.DefaultAPIPort,
	}

	serverNode := &types.Node{
		Name:  "k3d-test-server-0",
		Role:  types.ServerRole,
		Ports: nat.PortMap{},
	}

	loadBalancerNode := &types.Node{
		Name: "k3d-test-serverlb",
		Role: types.LoadBalancerRole,
		Ports: nat.PortMap{
			apiPort: {apiBinding},
		},
	}

	cluster := &types.Cluster{
		Name: "test",
		Nodes: []*types.Node{
			serverNode,
			loadBalancerNode,
		},
		ServerLoadBalancer: &types.Loadbalancer{
			Node: loadBalancerNode,
			Config: &types.LoadbalancerConfig{
				Ports: map[string][]string{
					fmt.Sprintf("%s.tcp", types.DefaultAPIPort): {serverNode.Name},
				},
				Settings: types.LoadBalancerSettings{WorkerConnections: types.DefaultLoadbalancerWorkerConnections},
			},
		},
		KubeAPI: &types.ExposureOpts{
			PortMapping: nat.PortMapping{
				Port:    apiPort,
				Binding: apiBinding,
			},
			Host: "0.0.0.0",
		},
	}

	return cluster
}
