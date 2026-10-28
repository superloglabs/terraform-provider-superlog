// Package provider is the Superlog Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/superloglabs/terraform-provider-superlog/internal/client"
)

type superlogProvider struct {
	version string
}

type providerModel struct {
	APIKey   types.String `tfsdk:"api_key"`
	Endpoint types.String `tfsdk:"endpoint"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &superlogProvider{version: version} }
}

func (p *superlogProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "superlog"
	resp.Version = p.version
}

func (p *superlogProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Superlog alerts, dashboards, and synthetic checks. Create an API key in Superlog under Settings → API keys. The provider acts as the member who created the key, in that key's workspace.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "A Superlog API key. Can also be set with the `SUPERLOG_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Superlog URL: `https://superlog.sh` for the EU region (the default) or `https://us.superlog.sh` for the US region. Can also be set with the `SUPERLOG_ENDPOINT` environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *superlogProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.APIKey.IsUnknown() || config.Endpoint.IsUnknown() {
		// Known at apply time; the provider is configured again then.
		return
	}
	apiKey := os.Getenv("SUPERLOG_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	endpoint := os.Getenv("SUPERLOG_ENDPOINT")
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing API key",
			"Set api_key in the provider block or the SUPERLOG_API_KEY environment variable. Create a key in Superlog under Settings → API keys.")
		return
	}
	c, err := client.New(endpoint, apiKey, "terraform-provider-superlog/"+p.version)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Invalid endpoint", err.Error())
		return
	}
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *superlogProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newAlertResource,
		newDashboardResource,
		newSyntheticCheckResource,
	}
}

func (p *superlogProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newSlackChannelDataSource,
	}
}
