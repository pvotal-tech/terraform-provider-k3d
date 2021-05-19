package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRegistry(t *testing.T) {
	//t.Skip("data source not yet implemented, remove this once you add your own code")

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRegistry,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.k3d_registry.foo", "name", regexp.MustCompile("^ba")),
				),
			},
		},
	})
}

const testAccDataSourceRegistry = `
resource "k3d_registry" "foo" {
  name = "bar"
}

data "k3d_registry" "foo" {
  depends_on = [ k3d_registry.foo ]

  name = k3d_registry.foo.name
}
`
