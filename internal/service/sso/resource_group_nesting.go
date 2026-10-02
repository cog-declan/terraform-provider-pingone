// Copyright © 2026 Ping Identity Corporation

package sso

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/patrickcping/pingone-go-sdk-v2/pingone/model"
	"github.com/pingidentity/terraform-provider-pingone/internal/framework"
	"github.com/pingidentity/terraform-provider-pingone/internal/framework/customtypes/pingonetypes"
	"github.com/pingidentity/terraform-provider-pingone/internal/framework/legacysdk"
	"github.com/pingidentity/terraform-provider-pingone/internal/sdk"
	"github.com/pingidentity/terraform-provider-pingone/internal/verify"
)

// Types
type GroupNestingResource serviceClientType

type GroupNestingResourceModel struct {
	Id            pingonetypes.ResourceIDValue `tfsdk:"id"`
	EnvironmentId pingonetypes.ResourceIDValue `tfsdk:"environment_id"`
	GroupId       pingonetypes.ResourceIDValue `tfsdk:"group_id"`
	NestedGroupId pingonetypes.ResourceIDValue `tfsdk:"nested_group_id"`
	Type          types.String                 `tfsdk:"type"`
}

// Framework interfaces
var (
	_ resource.Resource                = &GroupNestingResource{}
	_ resource.ResourceWithConfigure   = &GroupNestingResource{}
	_ resource.ResourceWithImportState = &GroupNestingResource{}
)

// New Object
func NewGroupNestingResource() resource.Resource {
	return &GroupNestingResource{}
}

// Metadata
func (r *GroupNestingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_nesting"
}

// Schema.
func (r *GroupNestingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {

	const attrMinLength = 1

	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Resource to create and manage PingOne group nesting.",

		Attributes: map[string]schema.Attribute{
			"id": framework.Attr_ID(),

			"environment_id": framework.Attr_LinkID(
				framework.SchemaAttributeDescriptionFromMarkdown("The ID of the environment to manage the group nesting in."),
			),

			"group_id": framework.Attr_LinkID(
				framework.SchemaAttributeDescriptionFromMarkdown("The ID of the parent group to assign the nested group to.  Members of the nested group (`nested_group_id`) become indirect members of this group, and inherit this group's permissions and application access."),
			),

			"nested_group_id": framework.Attr_LinkID(
				framework.SchemaAttributeDescriptionFromMarkdown("The ID of the group to configure as a nested group of the parent group (`group_id`).  Members of this group do not gain the permissions or application access of the parent group's members."),
			),

			"type": schema.StringAttribute{
				Description: framework.SchemaAttributeDescriptionFromMarkdown("The type of the group nesting.").Description,
				Computed:    true,
			},
		},
	}
}

func (r *GroupNestingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceConfig, ok := req.ProviderData.(legacysdk.ResourceType)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected the provider client, got: %T. Please report this issue to the provider maintainers.", req.ProviderData),
		)

		return
	}

	r.Client = resourceConfig.Client.API
	if r.Client == nil {
		resp.Diagnostics.AddError(
			"Client not initialised",
			"Expected the PingOne client, got nil.  Please report this issue to the provider maintainers.",
		)
		return
	}
}

func (r *GroupNestingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, state GroupNestingResourceModel

	if r.Client == nil || r.Client.ManagementAPIClient == nil {
		resp.Diagnostics.AddError(
			"Client not initialized",
			"Expected the PingOne client, got nil.  Please report this issue to the provider maintainers.")
		return
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the model for the API
	group := plan.expand()

	// Run the API call
	var response *management.GroupNesting
	resp.Diagnostics.Append(legacysdk.ParseResponse(
		ctx,

		func() (any, *http.Response, error) {
			fO, fR, fErr := r.Client.ManagementAPIClient.GroupsApi.CreateGroupNesting(ctx, plan.EnvironmentId.ValueString(), plan.NestedGroupId.ValueString()).GroupNesting(*group).Execute()
			return legacysdk.CheckEnvironmentExistsOnPermissionsError(ctx, r.Client.ManagementAPIClient, plan.EnvironmentId.ValueString(), fO, fR, fErr)
		},
		"CreateGroupNesting",
		legacysdk.DefaultCustomError,
		sdk.DefaultCreateReadRetryable,
		&response,
	)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the state to save
	state = plan

	// Save updated data into Terraform state
	resp.Diagnostics.Append(state.toState(response)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *GroupNestingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *GroupNestingResourceModel

	if r.Client == nil || r.Client.ManagementAPIClient == nil {
		resp.Diagnostics.AddError(
			"Client not initialized",
			"Expected the PingOne client, got nil.  Please report this issue to the provider maintainers.")
		return
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Run the API call
	response, d := r.readGroupNesting(ctx, data.EnvironmentId.ValueString(), data.NestedGroupId.ValueString(), data.GroupId.ValueString())
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If not found, check whether the nesting exists in the inverse direction (for example, created by an earlier provider version
	// that sent the group IDs to the API inverted).  The actual direction is stored in state so that Terraform plans to replace it.
	if response == nil {
		response, d = r.readGroupNesting(ctx, data.EnvironmentId.ValueString(), data.GroupId.ValueString(), data.NestedGroupId.ValueString())
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}

		if response != nil {
			resp.Diagnostics.AddWarning(
				"Inverted group nesting detected",
				fmt.Sprintf("The group \"%[1]s\" is nested within the group \"%[2]s\", which is the inverse of the configured relationship (\"%[2]s\" nested within \"%[1]s\").  Members of group \"%[1]s\" are indirect members of group \"%[2]s\" and inherit its access.  The group nesting will be replaced to match the configured direction.", data.GroupId.ValueString(), data.NestedGroupId.ValueString()),
			)

			data.GroupId, data.NestedGroupId = data.NestedGroupId, data.GroupId
		}
	}

	// Remove from state if resource is not found
	if response == nil {
		resp.Diagnostics.AddWarning("Requested resource not found", "The requested resource configuration cannot be found in the PingOne service.  If the requested resource is managed in Terraform's state, it may have been removed outside of Terraform.")
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(data.toState(response)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupNestingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *GroupNestingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *GroupNestingResourceModel

	if r.Client == nil || r.Client.ManagementAPIClient == nil {
		resp.Diagnostics.AddError(
			"Client not initialized",
			"Expected the PingOne client, got nil.  Please report this issue to the provider maintainers.")
		return
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Run the API call
	resp.Diagnostics.Append(legacysdk.ParseResponse(
		ctx,

		func() (any, *http.Response, error) {
			fR, fErr := r.Client.ManagementAPIClient.GroupsApi.DeleteGroupNesting(ctx, data.EnvironmentId.ValueString(), data.NestedGroupId.ValueString(), data.GroupId.ValueString()).Execute()
			return legacysdk.CheckEnvironmentExistsOnPermissionsError(ctx, r.Client.ManagementAPIClient, data.EnvironmentId.ValueString(), nil, fR, fErr)
		},
		"DeleteGroupNesting",
		legacysdk.CustomErrorResourceNotFoundWarning,
		nil,
		nil,
	)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *GroupNestingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	idComponents := []framework.ImportComponent{
		{
			Label:  "environment_id",
			Regexp: verify.P1ResourceIDRegexp,
		},
		{
			Label:  "group_id",
			Regexp: verify.P1ResourceIDRegexp,
		},
		{
			Label:  "nested_group_id",
			Regexp: verify.P1ResourceIDRegexp,
		},
	}

	attributes, err := framework.ParseImportID(req.ID, idComponents...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			err.Error(),
		)
		return
	}

	for _, idComponent := range idComponents {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(idComponent.Label), attributes[idComponent.Label])...)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), attributes["group_id"])...)
}

// readGroupNesting reads the nesting of nestedGroupID within groupID, returning a nil object if it is not found.
func (r *GroupNestingResource) readGroupNesting(ctx context.Context, environmentID, nestedGroupID, groupID string) (*management.GroupNesting, diag.Diagnostics) {
	var response *management.GroupNesting
	diags := legacysdk.ParseResponse(
		ctx,

		func() (any, *http.Response, error) {
			fO, fR, fErr := r.Client.ManagementAPIClient.GroupsApi.ReadOneGroupNesting(ctx, environmentID, nestedGroupID, groupID).Execute()
			return legacysdk.CheckEnvironmentExistsOnPermissionsError(ctx, r.Client.ManagementAPIClient, environmentID, fO, fR, fErr)
		},
		"ReadOneGroupNesting",
		func(r *http.Response, p1Error *model.P1Error) diag.Diagnostics {
			if (p1Error != nil && p1Error.GetCode() == "NOT_FOUND") || (r != nil && r.StatusCode == http.StatusNotFound) {
				return diag.Diagnostics{}
			}

			return nil
		},
		sdk.DefaultCreateReadRetryable,
		&response,
	)

	return response, diags
}

func (p *GroupNestingResourceModel) expand() *management.GroupNesting {

	data := management.NewGroupNesting(p.GroupId.ValueString())

	return data
}

func (p *GroupNestingResourceModel) toState(apiObject *management.GroupNesting) diag.Diagnostics {
	var diags diag.Diagnostics

	if apiObject == nil {
		diags.AddError(
			"Data object missing",
			"Cannot convert the data object to state as the data object is nil.  Please report this to the provider maintainers.",
		)

		return diags
	}

	p.Id = framework.PingOneResourceIDOkToTF(apiObject.GetIdOk())
	p.Type = framework.StringOkToTF(apiObject.GetTypeOk())
	p.GroupId = framework.PingOneResourceIDOkToTF(apiObject.GetIdOk())

	return diags
}
