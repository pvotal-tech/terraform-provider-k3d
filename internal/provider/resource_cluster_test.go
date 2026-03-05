package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestResourceClusterPortSchemaSupportsUpdate(t *testing.T) {
	resource := resourceCluster()
	portSchema, ok := resource.Schema["port"]
	if !ok {
		t.Fatalf("port schema not found")
	}

	if portSchema.ForceNew {
		t.Fatalf("expected port block to allow updates")
	}

	nested, ok := portSchema.Elem.(*schema.Resource)
	if !ok {
		t.Fatalf("unexpected element type for port schema: %T", portSchema.Elem)
	}

	for field, s := range nested.Schema {
		if s.ForceNew {
			t.Fatalf("expected field %s to allow updates", field)
		}
	}
}

func TestAccResourceCluster(t *testing.T) {
	//t.Skip("resource not yet implemented, remove this once you add your own code")

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceCluster,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"k3d_cluster.foo", "name", regexp.MustCompile("^ba")),
				),
			},
		},
	})
}

const testAccResourceCluster = `
resource "k3d_cluster" "foo" {
  name = "bar"
}
`
