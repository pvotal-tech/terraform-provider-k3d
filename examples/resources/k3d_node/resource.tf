resource "k3d_node" "mynode" {
  name = "mynode"

  cluster = "mycluster"
  image   = "rancher/k3s:v1.20.4-k3s1"
  memory  = "512M"
  role    = "agent"
}