package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceNode(t *testing.T) {
	//t.Skip("data source not yet implemented, remove this once you add your own code")

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNode,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.k3d_node.foo", "name", regexp.MustCompile("^ba")),
				),
			},
		},
	})
}

const testAccDataSourceNode = `
resource "k3d_cluster" "foo" {
  name = "bar"
}

resource "k3d_node" "foo" {
  name    = "bar"
  cluster = k3d_cluster.foo.name
}

data "k3d_node" "foo" {
  depends_on = [ k3d_node.foo ]

  name = k3d_node.foo.name
}
`
