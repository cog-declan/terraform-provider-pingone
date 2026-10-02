// Copyright © 2026 Ping Identity Corporation

package sso_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/pingidentity/terraform-provider-pingone/internal/acctest"
	acctestlegacysdk "github.com/pingidentity/terraform-provider-pingone/internal/acctest/legacysdk"
	baselegacysdk "github.com/pingidentity/terraform-provider-pingone/internal/acctest/service/base/legacysdk"
	"github.com/pingidentity/terraform-provider-pingone/internal/acctest/service/sso"
	client "github.com/pingidentity/terraform-provider-pingone/internal/client"
	"github.com/pingidentity/terraform-provider-pingone/internal/verify"
)

func TestAccGroupNesting_RemovalDrift(t *testing.T) {
	t.Parallel()

	resourceName := acctest.ResourceNameGen()
	resourceFullName := fmt.Sprintf("pingone_group_nesting.%s", resourceName)

	environmentName := acctest.ResourceNameGenEnvironment()

	name := resourceName

	licenseID := os.Getenv("PINGONE_LICENSE_ID")

	var nestedGroupID, groupID, environmentID string

	var p1Client *client.Client
	var ctx = context.Background()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheckNoTestAccFlaky(t)
			acctest.PreCheckClient(t)
			acctest.PreCheckNewEnvironment(t)
			acctest.PreCheckNoBeta(t)
			p1Client = acctestlegacysdk.PreCheckTestClient(ctx, t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             sso.GroupNesting_CheckDestroy,
		ErrorCheck:               acctest.ErrorCheck(t),
		Steps: []resource.TestStep{
			// Test removal of the resource
			{
				Config: testAccGroupNestingConfig_Full(resourceName, name),
				Check:  sso.GroupNesting_GetIDs(resourceFullName, &environmentID, &groupID, &nestedGroupID),
			},
			{
				PreConfig: func() {
					sso.GroupNesting_RemovalDrift_PreConfig(ctx, p1Client.API.ManagementAPIClient, t, environmentID, groupID, nestedGroupID)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			// Test removal of the group
			{
				Config: testAccGroupNestingConfig_Full(resourceName, name),
				Check:  sso.GroupNesting_GetIDs(resourceFullName, &environmentID, &groupID, &nestedGroupID),
			},
			{
				PreConfig: func() {
					sso.Group_RemovalDrift_PreConfig(ctx, p1Client.API.ManagementAPIClient, t, environmentID, groupID)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			// Test removal of the environment
			{
				Config: testAccGroupNestingConfig_NewEnv(environmentName, licenseID, resourceName, name),
				Check:  sso.GroupNesting_GetIDs(resourceFullName, &environmentID, &groupID, &nestedGroupID),
			},
			{
				PreConfig: func() {
					baselegacysdk.Environment_RemovalDrift_PreConfig(ctx, p1Client.API.ManagementAPIClient, t, environmentID)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccGroupNesting_Full(t *testing.T) {
	t.Parallel()

	resourceName := acctest.ResourceNameGen()
	resourceFullName := fmt.Sprintf("pingone_group_nesting.%s", resourceName)

	name := resourceName

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheckNoTestAccFlaky(t)
			acctest.PreCheckClient(t)
			acctest.PreCheckNoBeta(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             sso.GroupNesting_CheckDestroy,
		ErrorCheck:               acctest.ErrorCheck(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupNestingConfig_Full(resourceName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(resourceFullName, "id", verify.P1ResourceIDRegexpFullString),
					resource.TestMatchResourceAttr(resourceFullName, "environment_id", verify.P1ResourceIDRegexpFullString),
					resource.TestMatchResourceAttr(resourceFullName, "group_id", verify.P1ResourceIDRegexpFullString),
					resource.TestMatchResourceAttr(resourceFullName, "nested_group_id", verify.P1ResourceIDRegexpFullString),
					resource.TestCheckResourceAttrPair(resourceFullName, "id", resourceFullName, "group_id"),
					resource.TestCheckResourceAttr(resourceFullName, "type", "DIRECT"),
					sso.GroupNesting_CheckDirection(resourceFullName),
				),
			},
			// Test importing the resource
			{
				ResourceName: resourceFullName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						rs, ok := s.RootModule().Resources[resourceFullName]
						if !ok {
							return "", fmt.Errorf("resource not found: %s", resourceFullName)
						}

						return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["environment_id"], rs.Primary.Attributes["group_id"], rs.Primary.Attributes["nested_group_id"]), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGroupNesting_InvertedDirectionDrift(t *testing.T) {
	t.Parallel()

	resourceName := acctest.ResourceNameGen()
	resourceFullName := fmt.Sprintf("pingone_group_nesting.%s", resourceName)

	name := resourceName

	var nestedGroupID, groupID, environmentID string

	var p1Client *client.Client
	var ctx = context.Background()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheckNoTestAccFlaky(t)
			acctest.PreCheckClient(t)
			acctest.PreCheckNoBeta(t)
			p1Client = acctestlegacysdk.PreCheckTestClient(ctx, t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             sso.GroupNesting_CheckDestroy,
		ErrorCheck:               acctest.ErrorCheck(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupNestingConfig_Full(resourceName, name),
				Check: resource.ComposeTestCheckFunc(
					sso.GroupNesting_GetIDs(resourceFullName, &environmentID, &groupID, &nestedGroupID),
					sso.GroupNesting_CheckDirection(resourceFullName),
				),
			},
			// Replace the nesting with the inverse direction outside of Terraform
			{
				PreConfig: func() {
					sso.GroupNesting_Invert_PreConfig(ctx, p1Client.API.ManagementAPIClient, t, environmentID, groupID, nestedGroupID)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPtr(resourceFullName, "group_id", &nestedGroupID),
					resource.TestCheckResourceAttrPtr(resourceFullName, "nested_group_id", &groupID),
				),
			},
			// The inverted nesting is replaced with the configured direction
			{
				Config: testAccGroupNestingConfig_Full(resourceName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPtr(resourceFullName, "group_id", &groupID),
					resource.TestCheckResourceAttrPtr(resourceFullName, "nested_group_id", &nestedGroupID),
					sso.GroupNesting_CheckDirection(resourceFullName),
				),
			},
		},
	})
}

func TestAccGroupNesting_BadParameters(t *testing.T) {
	t.Parallel()

	resourceName := acctest.ResourceNameGen()
	resourceFullName := fmt.Sprintf("pingone_group_nesting.%s", resourceName)

	name := resourceName

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheckNoTestAccFlaky(t)
			acctest.PreCheckClient(t)
			acctest.PreCheckNoBeta(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             sso.GroupNesting_CheckDestroy,
		ErrorCheck:               acctest.ErrorCheck(t),
		Steps: []resource.TestStep{
			// Configure
			{
				Config: testAccGroupNestingConfig_Full(resourceName, name),
			},
			// Errors
			{
				ResourceName: resourceFullName,
				ImportState:  true,
				ExpectError:  regexp.MustCompile(`Unexpected Import Identifier`),
			},
			{
				ResourceName:  resourceFullName,
				ImportStateId: "/",
				ImportState:   true,
				ExpectError:   regexp.MustCompile(`Unexpected Import Identifier`),
			},
			{
				ResourceName:  resourceFullName,
				ImportStateId: "badformat/badformat/badformat",
				ImportState:   true,
				ExpectError:   regexp.MustCompile(`Unexpected Import Identifier`),
			},
		},
	})
}

func testAccGroupNestingConfig_NewEnv(environmentName, licenseID, resourceName, name string) string {
	return fmt.Sprintf(`
		%[1]s

resource "pingone_group" "%[3]s" {
  environment_id = pingone_environment.%[2]s.id

  name = "%[4]s"
}

resource "pingone_group" "%[3]s-nesting" {
  environment_id = pingone_environment.%[2]s.id

  name = "%[4]s-nesting"
}

resource "pingone_group_nesting" "%[3]s" {
  environment_id  = pingone_environment.%[2]s.id
  group_id        = pingone_group.%[3]s.id
  nested_group_id = pingone_group.%[3]s-nesting.id
}`, acctestlegacysdk.MinimalSandboxEnvironment(environmentName, licenseID), environmentName, resourceName, name)
}

func testAccGroupNestingConfig_Full(resourceName, name string) string {
	return fmt.Sprintf(`
		%[1]s

resource "pingone_group" "%[2]s" {
  environment_id = data.pingone_environment.general_test.id

  name = "%[3]s"
}

resource "pingone_group" "%[2]s-nesting" {
  environment_id = data.pingone_environment.general_test.id

  name = "%[3]s-nesting"
}

resource "pingone_group_nesting" "%[2]s" {
  environment_id  = data.pingone_environment.general_test.id
  group_id        = pingone_group.%[2]s.id
  nested_group_id = pingone_group.%[2]s-nesting.id
}`, acctest.GenericSandboxEnvironment(), resourceName, name)
}
