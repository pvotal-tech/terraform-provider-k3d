---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "k3d_node Data Source - terraform-provider-k3d"
subcategory: ""
description: |-
  Node data source in k3d.
---

# k3d_node (Data Source)

Node data source in k3d.

## Example Usage

```terraform
data "k3d_node" "mynode" {
  name = "mynode"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Node name.

### Read-Only

- `cluster` (String) Select the cluster that the node shall connect to.
- `id` (String) The ID of this resource.
- `role` (String) Specify node role [server, agent].


