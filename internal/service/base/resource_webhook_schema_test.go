// Copyright © 2026 Ping Identity Corporation

package base_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/pingidentity/terraform-provider-pingone/internal/service/base"
)

func TestWebhookResourceSchema_HeadersSensitive(t *testing.T) {
	ctx := context.Background()

	r := base.NewWebhookResource()

	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", resp.Diagnostics)
	}

	for _, attrName := range []string{"http_endpoint_headers", "connection_details_headers"} {
		attr, ok := resp.Schema.Attributes[attrName]
		if !ok {
			t.Fatalf("attribute %q not found in schema", attrName)
		}

		if !attr.IsSensitive() {
			t.Errorf("attribute %q must be marked Sensitive", attrName)
		}
	}

	upgrader, ok := r.(resource.ResourceWithUpgradeState)
	if !ok {
		t.Fatal("webhook resource does not implement ResourceWithUpgradeState")
	}

	priorSchema := upgrader.UpgradeState(ctx)[0].PriorSchema
	if priorSchema == nil {
		t.Fatal("webhook v0 state upgrader has no prior schema")
	}

	attr, ok := priorSchema.Attributes["http_endpoint_headers"]
	if !ok {
		t.Fatal("attribute \"http_endpoint_headers\" not found in v0 prior schema")
	}

	if !attr.IsSensitive() {
		t.Error("attribute \"http_endpoint_headers\" must be marked Sensitive in v0 prior schema")
	}
}
