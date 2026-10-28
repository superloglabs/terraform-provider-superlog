package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/superloglabs/terraform-provider-superlog/internal/client"
)

var _ datasource.DataSourceWithConfigure = &slackChannelDataSource{}

type slackChannelDataSource struct {
	client *client.Client
}

type slackChannelModel struct {
	ChannelID            types.String `tfsdk:"channel_id"`
	IntegrationAccountID types.String `tfsdk:"integration_account_id"`
	Name                 types.String `tfsdk:"name"`
	Private              types.Bool   `tfsdk:"private"`
	Workspace            types.String `tfsdk:"workspace"`
}

func newSlackChannelDataSource() datasource.DataSource { return &slackChannelDataSource{} }

func (d *slackChannelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_slack_channel"
}

func (d *slackChannelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Slack channel alerts and synthetic checks can post to: a public channel, or a private channel the Superlog app is in.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The channel name, with or without `#`.",
				Required:            true,
			},
			"workspace": schema.StringAttribute{
				MarkdownDescription: "The Slack workspace's name. Needed when more than one connected workspace has a channel with this name.",
				Optional:            true,
				Computed:            true,
			},
			"channel_id":             schema.StringAttribute{Computed: true},
			"integration_account_id": schema.StringAttribute{Computed: true},
			"private":                schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *slackChannelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req.ProviderData, &resp.Diagnostics)
}

func (d *slackChannelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config slackChannelModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	channels, err := d.client.ListSlackChannels(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Slack channels", err.Error())
		return
	}
	name := strings.TrimPrefix(config.Name.ValueString(), "#")
	var matches []client.SlackChannel
	for _, channel := range channels {
		if channel.Name != name {
			continue
		}
		if !config.Workspace.IsNull() && !config.Workspace.IsUnknown() && channel.AccountName != config.Workspace.ValueString() {
			continue
		}
		matches = append(matches, channel)
	}
	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Slack channel not found", fmt.Sprintf(
			"No connected Slack workspace has a channel #%s Superlog can post to. For a private channel, invite the Superlog app first. New channels can take a few minutes to appear.", name))
		return
	case 1:
	default:
		resp.Diagnostics.AddError("More than one Slack channel matches", fmt.Sprintf(
			"More than one connected Slack workspace has #%s. Set workspace to choose one.", name))
		return
	}
	channel := matches[0]
	resp.Diagnostics.Append(resp.State.Set(ctx, slackChannelModel{
		ChannelID:            types.StringValue(channel.ChannelID),
		IntegrationAccountID: types.StringValue(channel.IntegrationAccountID),
		Name:                 config.Name,
		Private:              types.BoolValue(channel.Private),
		Workspace:            types.StringValue(channel.AccountName),
	})...)
}
