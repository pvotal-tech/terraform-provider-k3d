package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRegistry(t *testing.T) {
	//t.Skip("resource not yet implemented, remove this once you add your own code")

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRegistry,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"k3d_registry.foo", "name", regexp.MustCompile("^ba")),
				),
			},
		},
	})
}

const testAccResourceRegistry = `
resource "k3d_registry" "foo" {
  name = "bar"
}
`
