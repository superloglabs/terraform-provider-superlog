package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/superloglabs/terraform-provider-superlog/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = &alertResource{}
	_ resource.ResourceWithImportState = &alertResource{}
)

type alertResource struct {
	client *client.Client
}

type attributeFilterModel struct {
	Key   types.String `tfsdk:"key"`
	Op    types.String `tfsdk:"op"`
	Value types.String `tfsdk:"value"`
}

type alertFilterModel struct {
	Attributes    []attributeFilterModel `tfsdk:"attributes"`
	MetricKind    types.String           `tfsdk:"metric_kind"`
	MetricUnit    types.String           `tfsdk:"metric_unit"`
	MinDurationMs types.Float64          `tfsdk:"min_duration_ms"`
	Service       types.String           `tfsdk:"service"`
	Severity      types.String           `tfsdk:"severity"`
	SpanName      types.String           `tfsdk:"span_name"`
	StatusCode    types.String           `tfsdk:"status_code"`
}

type alertModel struct {
	Aggregation               types.String      `tfsdk:"aggregation"`
	Comparator                types.String      `tfsdk:"comparator"`
	Enabled                   types.Bool        `tfsdk:"enabled"`
	EvaluationIntervalSeconds types.Int64       `tfsdk:"evaluation_interval_seconds"`
	Field                     types.String      `tfsdk:"field"`
	Filter                    *alertFilterModel `tfsdk:"filter"`
	GroupBy                   types.String      `tfsdk:"group_by"`
	GroupMode                 types.String      `tfsdk:"group_mode"`
	ID                        types.String      `tfsdk:"id"`
	MetricName                types.String      `tfsdk:"metric_name"`
	Name                      types.String      `tfsdk:"name"`
	Slack                     *slackModel       `tfsdk:"slack"`
	Source                    types.String      `tfsdk:"source"`
	Threshold                 types.Float64     `tfsdk:"threshold"`
	URL                       types.String      `tfsdk:"url"`
	WindowMinutes             types.Int64       `tfsdk:"window_minutes"`
}

func newAlertResource() resource.Resource { return &alertResource{} }

func (r *alertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert"
}

func attributeFiltersAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "Attribute filters. A key may start with `resource.`, `span.`, `log.`, or `field.`; a bare key is a resource attribute.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"key": schema.StringAttribute{Required: true},
				"op": schema.StringAttribute{
					MarkdownDescription: "`eq` (the default), `neq`, or `not_contains`.",
					Optional:            true,
					Validators:          []validator.String{stringvalidator.OneOf("eq", "neq", "not_contains")},
				},
				"value": schema.StringAttribute{Required: true},
			},
		},
	}
}

func slackAttribute(what string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "The Slack channel the " + what + " posts to. Look it up with the `superlog_slack_channel` data source. Without one, the " + what + " only records its history in Superlog.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"channel_id":             schema.StringAttribute{Required: true, MarkdownDescription: "The Slack channel ID."},
			"integration_account_id": schema.StringAttribute{Required: true, MarkdownDescription: "The Superlog Slack integration the channel belongs to."},
		},
	}
}

func (r *alertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A threshold alert on logs, spans, or a metric. While the value is past the threshold, the alert posts once to its Slack channel and posts the recovery in the same thread.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The alert in Superlog.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "A disabled alert is not checked. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"evaluation_interval_seconds": schema.Int64Attribute{
				MarkdownDescription: "How often the alert is checked, from 60 to 3600 seconds. Defaults to 60.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
				Validators:          []validator.Int64{int64validator.Between(60, 3600)},
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "`logs`, `traces` (spans), or `metric`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("logs", "traces", "metric")},
			},
			"aggregation": schema.StringAttribute{
				MarkdownDescription: "`count` counts matching logs or spans; `count_distinct`, `sum`, `avg`, `min`, `max`, `p50`, `p90`, `p95`, and `p99` read `field`. Metrics use `sum` or `avg`.",
				Required:            true,
				Validators: []validator.String{stringvalidator.OneOf(
					"count", "count_distinct", "sum", "avg", "min", "max", "p50", "p90", "p95", "p99",
				)},
			},
			"field": schema.StringAttribute{
				MarkdownDescription: "What a log or span aggregation other than `count` reads: `duration`, an attribute key, or an AI usage field such as `ai:cost`.",
				Optional:            true,
			},
			"metric_name": schema.StringAttribute{
				MarkdownDescription: "The metric to read when `source` is `metric`.",
				Optional:            true,
			},
			"comparator": schema.StringAttribute{
				MarkdownDescription: "`gt` fires when the value is above `threshold`, `lt` when it is below.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("gt", "lt")},
			},
			"threshold": schema.Float64Attribute{Required: true},
			"window_minutes": schema.Int64Attribute{
				MarkdownDescription: "How many minutes of data each check reads, from 1 to 1440. Defaults to 5.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(5),
				Validators:          []validator.Int64{int64validator.Between(1, 1440)},
			},
			"group_mode": schema.StringAttribute{
				MarkdownDescription: "`single` checks one total. `per_group` checks each `group_by` value and fires for each one separately. Defaults to `single`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("single"),
				Validators:          []validator.String{stringvalidator.OneOf("single", "per_group")},
			},
			"group_by": schema.StringAttribute{
				MarkdownDescription: "The attribute key to split by when `group_mode` is `per_group`, such as `service.name`.",
				Optional:            true,
			},
			"filter": schema.SingleNestedAttribute{
				MarkdownDescription: "Narrows the data the alert reads.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"attributes":      attributeFiltersAttribute(),
					"service":         schema.StringAttribute{Optional: true},
					"severity":        schema.StringAttribute{Optional: true, MarkdownDescription: "Log severity, such as `ERROR`."},
					"span_name":       schema.StringAttribute{Optional: true},
					"status_code":     schema.StringAttribute{Optional: true, MarkdownDescription: "Span status code, such as `ERROR`."},
					"min_duration_ms": schema.Float64Attribute{Optional: true, MarkdownDescription: "Only spans at least this long."},
					"metric_kind": schema.StringAttribute{
						MarkdownDescription: "Picks one metric when its name is sent as several kinds.",
						Optional:            true,
						Validators:          []validator.String{stringvalidator.OneOf("gauge", "sum", "histogram", "exponential_histogram", "summary")},
					},
					"metric_unit": schema.StringAttribute{Optional: true, MarkdownDescription: "Picks one metric when its name is sent with several units."},
				},
			},
			"slack": slackAttribute("alert"),
		},
	}
}

func (r *alertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics)
}

func alertInput(model alertModel) client.AlertInput {
	input := client.AlertInput{
		Aggregation:               model.Aggregation.ValueString(),
		Comparator:                model.Comparator.ValueString(),
		Enabled:                   model.Enabled.ValueBool(),
		EvaluationIntervalSeconds: model.EvaluationIntervalSeconds.ValueInt64(),
		Field:                     stringPointer(model.Field),
		GroupBy:                   stringPointer(model.GroupBy),
		GroupMode:                 model.GroupMode.ValueString(),
		MetricName:                stringPointer(model.MetricName),
		Name:                      model.Name.ValueString(),
		Slack:                     slackTarget(model.Slack),
		Source:                    model.Source.ValueString(),
		Threshold:                 model.Threshold.ValueFloat64(),
		WindowMinutes:             model.WindowMinutes.ValueInt64(),
	}
	if filter := model.Filter; filter != nil {
		input.Filter = client.AlertFilter{
			MetricKind:    stringPointer(filter.MetricKind),
			MetricUnit:    stringPointer(filter.MetricUnit),
			ResourceAttrs: attributeFilters(filter.Attributes),
			Service:       stringPointer(filter.Service),
			Severity:      stringPointer(filter.Severity),
			SpanName:      stringPointer(filter.SpanName),
			StatusCode:    stringPointer(filter.StatusCode),
		}
		if !filter.MinDurationMs.IsNull() && !filter.MinDurationMs.IsUnknown() {
			value := filter.MinDurationMs.ValueFloat64()
			input.Filter.MinDurationMs = &value
		}
	}
	return input
}

func attributeFilters(models []attributeFilterModel) []client.AttributeFilter {
	filters := make([]client.AttributeFilter, 0, len(models))
	for _, model := range models {
		filters = append(filters, client.AttributeFilter{
			Key:   model.Key.ValueString(),
			Op:    stringPointer(model.Op),
			Value: model.Value.ValueString(),
		})
	}
	return filters
}

func attributeFiltersFromAPI(filters []client.AttributeFilter) []attributeFilterModel {
	if len(filters) == 0 {
		return nil
	}
	models := make([]attributeFilterModel, 0, len(filters))
	for _, filter := range filters {
		models = append(models, attributeFilterModel{
			Key:   types.StringValue(filter.Key),
			Op:    stringValue(filter.Op),
			Value: types.StringValue(filter.Value),
		})
	}
	return models
}

func alertFromAPI(alert *client.Alert) alertModel {
	model := alertModel{
		Aggregation:               types.StringValue(alert.Aggregation),
		Comparator:                types.StringValue(alert.Comparator),
		Enabled:                   types.BoolValue(alert.Enabled),
		EvaluationIntervalSeconds: types.Int64Value(alert.EvaluationIntervalSeconds),
		Field:                     stringValue(alert.Field),
		GroupBy:                   stringValue(alert.GroupBy),
		GroupMode:                 types.StringValue(alert.GroupMode),
		ID:                        types.StringValue(alert.ID),
		MetricName:                stringValue(alert.MetricName),
		Name:                      types.StringValue(alert.Name),
		Slack:                     slackFromAPI(alert.Slack),
		Source:                    types.StringValue(alert.Source),
		Threshold:                 types.Float64Value(alert.Threshold),
		URL:                       types.StringValue(alert.URL),
		WindowMinutes:             types.Int64Value(alert.WindowMinutes),
	}
	filter := alert.Filter
	empty := filter.MetricKind == nil && filter.MetricUnit == nil && filter.MinDurationMs == nil &&
		len(filter.ResourceAttrs) == 0 && filter.Service == nil && filter.Severity == nil &&
		filter.SpanName == nil && filter.StatusCode == nil
	if !empty {
		model.Filter = &alertFilterModel{
			Attributes:    attributeFiltersFromAPI(filter.ResourceAttrs),
			MetricKind:    stringValue(filter.MetricKind),
			MetricUnit:    stringValue(filter.MetricUnit),
			MinDurationMs: types.Float64Null(),
			Service:       stringValue(filter.Service),
			Severity:      stringValue(filter.Severity),
			SpanName:      stringValue(filter.SpanName),
			StatusCode:    stringValue(filter.StatusCode),
		}
		if filter.MinDurationMs != nil {
			model.Filter.MinDurationMs = types.Float64Value(*filter.MinDurationMs)
		}
	}
	return model
}

func (r *alertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alert, err := r.client.CreateAlert(ctx, alertInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create alert", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, alertFromAPI(alert))...)
}

func (r *alertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alert, err := r.client.GetAlert(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read alert", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, alertFromAPI(alert))...)
}

func (r *alertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state alertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alert, err := r.client.UpdateAlert(ctx, state.ID.ValueString(), alertInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to update alert", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, alertFromAPI(alert))...)
}

func (r *alertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAlert(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete alert", err.Error())
	}
}

func (r *alertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
