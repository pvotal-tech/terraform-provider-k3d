package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceCluster(t *testing.T) {
	//t.Skip("data source not yet implemented, remove this once you add your own code")

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceCluster,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.k3d_cluster.foo", "name", regexp.MustCompile("^ba")),
				),
			},
		},
	})
}

const testAccDataSourceCluster = `
resource "k3d_cluster" "foo" {
  name = "bar"
}

data "k3d_cluster" "foo" {
  depends_on = [ k3d_cluster.foo ]

  name = k3d_cluster.foo.name
}
`
