// Copyright © 2026 Ping Identity Corporation

package davinci_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/pingidentity/terraform-provider-pingone/internal/service/davinci"
)

func davinciApplicationSchemaAttributes(t *testing.T, ds datasource.DataSource) map[string]schema.Attribute {
	t.Helper()

	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", resp.Diagnostics)
	}

	return resp.Schema.Attributes
}

func assertDavinciApplicationSecretsSensitive(t *testing.T, attrs map[string]schema.Attribute) {
	t.Helper()

	cases := []struct {
		parent string
		child  string
	}{
		{"api_key", "value"},
		{"oauth", "client_secret"},
	}

	for _, c := range cases {
		parent, ok := attrs[c.parent].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("expected %s to be a SingleNestedAttribute", c.parent)
		}

		child, ok := parent.Attributes[c.child]
		if !ok {
			t.Fatalf("expected %s.%s to exist", c.parent, c.child)
		}

		if !child.IsSensitive() {
			t.Errorf("expected %s.%s to be sensitive", c.parent, c.child)
		}
	}
}

func TestDavinciApplicationDataSource_SecretsSensitive(t *testing.T) {
	assertDavinciApplicationSecretsSensitive(t, davinciApplicationSchemaAttributes(t, davinci.NewDavinciApplicationDataSource()))
}

func TestDavinciApplicationsDataSource_SecretsSensitive(t *testing.T) {
	attrs := davinciApplicationSchemaAttributes(t, davinci.NewDavinciApplicationsDataSource())

	applications, ok := attrs["davinci_applications"].(schema.SetNestedAttribute)
	if !ok {
		t.Fatalf("expected davinci_applications to be a SetNestedAttribute")
	}

	assertDavinciApplicationSecretsSensitive(t, applications.NestedObject.Attributes)
}
