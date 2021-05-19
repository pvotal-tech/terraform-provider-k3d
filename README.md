# Terraform Provider for k3d

This provider manages [k3d](https://k3d.io) Kubernetes clusters.

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.16

## Using the provider

The provider configuration follows more or less k3d's [config file](https://k3d.io/usage/configfile/) format:

```
provider "k3d" {}

resource "k3d_cluster" "mycluster" {
  name    = "mycluster"
  servers = 1
  agents  = 2

  kube_api {
    host      = "myhost.my.domain"
    host_ip   = "127.0.0.1"
    host_port = 6445
  }

  image   = "rancher/k3s:v1.20.4-k3s1"
  network = "my-custom-net"
  token   = "superSecretToken"

  volume {
    source      = "/my/host/path"
    destination = "/path/in/node"
    node_filters = [
      "server[0]",
      "agent[*]",
    ]
  }

  port {
    host_port      = 8080
    container_port = 80
    node_filters = [
      "loadbalancer",
    ]
  }

  label {
    key   = "foo"
    value = "bar"
    node_filters = [
      "agent[1]",
    ]
  }

  env {
    key   = "bar"
    value = "baz"
    node_filters = [
      "server[0]",
    ]
  }

  registries {
    create = true
    use = [
      "k3d-myotherregistry:5000"
    ]
    config = <<EOF
mirrors:
  "my.company.registry":
    endpoint:
      - http://my.company.registry:5000
EOF
  }

  k3d {
    disable_load_balancer     = false
    disable_image_volume      = false
    disable_host_ip_injection = false
  }

  k3s {
    extra_server_args = [
      "--tls-san=my.host.domain",
    ]
    extra_agent_args = []
  }

  kubeconfig {
    update_default_kubeconfig = true
    switch_current_context    = true
  }

  runtime {
    gpu_request = "all"
  }
}
```

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command: 
```sh
$ go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
