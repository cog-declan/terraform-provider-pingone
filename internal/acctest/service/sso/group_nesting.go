// Copyright © 2026 Ping Identity Corporation

package sso

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/pingidentity/terraform-provider-pingone/internal/acctest"
	"github.com/pingidentity/terraform-provider-pingone/internal/acctest/legacysdk"
)

func GroupNesting_CheckDestroy(s *terraform.State) error {
	var ctx = context.Background()

	p1Client, err := legacysdk.TestClient(ctx)

	if err != nil {
		return err
	}

	apiClient := p1Client.API.ManagementAPIClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pingone_group_nesting" {
			continue
		}

		shouldContinue, err := legacysdk.CheckParentEnvironmentDestroy(ctx, p1Client.API.ManagementAPIClient, rs.Primary.Attributes["environment_id"])
		if err != nil {
			return err
		}

		if shouldContinue {
			continue
		}

		_, r, err := apiClient.GroupsApi.ReadOneGroupNesting(ctx, rs.Primary.Attributes["environment_id"], rs.Primary.Attributes["nested_group_id"], rs.Primary.Attributes["group_id"]).Execute()

		shouldContinue, err = acctest.CheckForResourceDestroy(r, err)
		if err != nil {
			return err
		}

		if shouldContinue {
			continue
		}

		return fmt.Errorf("PingOne Group Nesting Instance %s still exists", rs.Primary.ID)
	}

	return nil
}

func GroupNesting_GetIDs(resourceName string, environmentID, groupID, nestedGroupID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if nestedGroupID != nil {
			*nestedGroupID = rs.Primary.Attributes["nested_group_id"]
		}

		if groupID != nil {
			*groupID = rs.Primary.Attributes["group_id"]
		}

		if environmentID != nil {
			*environmentID = rs.Primary.Attributes["environment_id"]
		}

		return nil
	}
}

func GroupNesting_RemovalDrift_PreConfig(ctx context.Context, apiClient *management.APIClient, t *testing.T, environmentID, groupID, nestedGroupID string) {
	if environmentID == "" || groupID == "" || nestedGroupID == "" {
		t.Fatalf("One of environment ID, group ID or nested group ID cannot be determined. Environment ID: %s, Group ID: %s, Nested Group ID: %s", environmentID, groupID, nestedGroupID)
	}

	_, err := apiClient.GroupsApi.DeleteGroupNesting(ctx, environmentID, nestedGroupID, groupID).Execute()
	if err != nil {
		t.Fatalf("Failed to delete group nesting: %v", err)
	}
}

// GroupNesting_Invert_PreConfig replaces the nesting of nestedGroupID within groupID with the inverse nesting (groupID within nestedGroupID).
func GroupNesting_Invert_PreConfig(ctx context.Context, apiClient *management.APIClient, t *testing.T, environmentID, groupID, nestedGroupID string) {
	GroupNesting_RemovalDrift_PreConfig(ctx, apiClient, t, environmentID, groupID, nestedGroupID)

	_, _, err := apiClient.GroupsApi.CreateGroupNesting(ctx, environmentID, groupID).GroupNesting(*management.NewGroupNesting(nestedGroupID)).Execute()
	if err != nil {
		t.Fatalf("Failed to create inverted group nesting: %v", err)
	}
}

// GroupNesting_CheckDirection verifies, using the PingOne memberOfGroups API, that the resource's nested_group_id is a member of its group_id, and not the inverse.
func GroupNesting_CheckDirection(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var ctx = context.Background()

		p1Client, err := legacysdk.TestClient(ctx)
		if err != nil {
			return err
		}

		apiClient := p1Client.API.ManagementAPIClient

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		environmentID := rs.Primary.Attributes["environment_id"]
		groupID := rs.Primary.Attributes["group_id"]
		nestedGroupID := rs.Primary.Attributes["nested_group_id"]

		memberOf := func(childGroupID, parentGroupID string) (bool, error) {
			pagedIterator := apiClient.GroupsApi.ReadGroupNesting(ctx, environmentID, childGroupID).Execute()

			for pageCursor, err := range pagedIterator {
				if err != nil {
					return false, err
				}

				if pageCursor.EntityArray == nil || pageCursor.EntityArray.Embedded == nil {
					continue
				}

				for _, groupMembership := range pageCursor.EntityArray.Embedded.GetGroupMemberships() {
					if groupMembership.GetId() == parentGroupID {
						return true, nil
					}
				}
			}

			return false, nil
		}

		if ok, err := memberOf(nestedGroupID, groupID); err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("expected group %s (nested_group_id) to be a member of group %s (group_id)", nestedGroupID, groupID)
		}

		if ok, err := memberOf(groupID, nestedGroupID); err != nil {
			return err
		} else if ok {
			return fmt.Errorf("group %s (group_id) is unexpectedly a member of group %s (nested_group_id)", groupID, nestedGroupID)
		}

		return nil
	}
}
