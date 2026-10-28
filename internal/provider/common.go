package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/superloglabs/terraform-provider-superlog/internal/client"
)

// configuredClient takes the client the provider configured for a resource
// or data source.
func configuredClient(data any, diagnostics *diag.Diagnostics) *client.Client {
	if data == nil {
		return nil
	}
	c, ok := data.(*client.Client)
	if !ok {
		diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", data))
		return nil
	}
	return c
}

func stringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueString()
	return &v
}

func stringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// slackModel is where an alert or check posts.
type slackModel struct {
	ChannelID            types.String `tfsdk:"channel_id"`
	IntegrationAccountID types.String `tfsdk:"integration_account_id"`
}

func slackTarget(model *slackModel) *client.SlackTarget {
	if model == nil {
		return nil
	}
	return &client.SlackTarget{
		ChannelID:            model.ChannelID.ValueString(),
		IntegrationAccountID: model.IntegrationAccountID.ValueString(),
	}
}

func slackFromAPI(target *client.SlackTarget) *slackModel {
	if target == nil {
		return nil
	}
	return &slackModel{
		ChannelID:            types.StringValue(target.ChannelID),
		IntegrationAccountID: types.StringValue(target.IntegrationAccountID),
	}
}
