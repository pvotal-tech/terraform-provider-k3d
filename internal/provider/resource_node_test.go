package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceNode(t *testing.T) {
	//t.Skip("resource not yet implemented, remove this once you add your own code")

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNode,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"k3d_node.foo", "name", regexp.MustCompile("^ba")),
				),
			},
		},
	})
}

const testAccResourceNode = `
resource "k3d_cluster" "foo" {
  name = "bar"
}

resource "k3d_node" "foo" {
  name    = "bar"
  cluster = k3d_cluster.foo.name
}
`
