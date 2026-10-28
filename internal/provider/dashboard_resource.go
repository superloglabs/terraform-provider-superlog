package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/superloglabs/terraform-provider-superlog/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = &dashboardResource{}
	_ resource.ResourceWithImportState = &dashboardResource{}
)

type dashboardResource struct {
	client *client.Client
}

type variableModel struct {
	AttributeKey types.String `tfsdk:"attribute_key"`
	DefaultValue types.String `tfsdk:"default_value"`
	Label        types.String `tfsdk:"label"`
	Multi        types.Bool   `tfsdk:"multi"`
	Name         types.String `tfsdk:"name"`
	Options      []string     `tfsdk:"options"`
}

type widgetModel struct {
	Config widgetConfig `tfsdk:"config"`
	ID     types.String `tfsdk:"id"`
	Layout types.Object `tfsdk:"layout"`
	Title  types.String `tfsdk:"title"`
	Type   types.String `tfsdk:"type"`
}

type layoutModel struct {
	H types.Int64 `tfsdk:"h"`
	W types.Int64 `tfsdk:"w"`
	X types.Int64 `tfsdk:"x"`
	Y types.Int64 `tfsdk:"y"`
}

var layoutTypes = map[string]attr.Type{"h": types.Int64Type, "w": types.Int64Type, "x": types.Int64Type, "y": types.Int64Type}

type dashboardModel struct {
	ID        types.String    `tfsdk:"id"`
	Name      types.String    `tfsdk:"name"`
	URL       types.String    `tfsdk:"url"`
	Variables []variableModel `tfsdk:"variables"`
	Widgets   []widgetModel   `tfsdk:"widgets"`
}

func newDashboardResource() resource.Resource { return &dashboardResource{} }

func (r *dashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *dashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	variableAttributes := map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Widgets use the selected value when a filter value is `$name` or `${name}`.",
			Required:            true,
		},
		"label":         schema.StringAttribute{Optional: true},
		"attribute_key": schema.StringAttribute{Optional: true, MarkdownDescription: "The attribute the variable usually filters, such as `deployment.environment`."},
		"options": schema.ListAttribute{
			MarkdownDescription: "The choices. Leave it empty to type a value.",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
		},
		"default_value": schema.StringAttribute{Optional: true, MarkdownDescription: "Must be one of `options` when there are any."},
		"multi":         schema.BoolAttribute{Optional: true, MarkdownDescription: "Allows several values. Selecting none matches every value."},
	}
	widgetAttributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"title": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 200)}},
		"type": schema.StringAttribute{
			MarkdownDescription: "`timeseries_count` counts or aggregates logs or spans, `timeseries_metric` charts one metric, `log_table` and `trace_table` list matching logs or traces, and `markdown` shows text.",
			Required:            true,
			Validators: []validator.String{stringvalidator.OneOf(
				"timeseries_count", "timeseries_metric", "log_table", "trace_table", "markdown",
			)},
		},
		"config": schema.StringAttribute{
			MarkdownDescription: "The widget's settings as JSON; use `jsonencode`. For example `{ source = \"logs\", groupBy = \"service.name\" }`, `{ metricName = \"http.server.duration\", aggregation = \"p95\" }`, or `{ markdown = \"...\" }`. Filters go in `filter.resourceAttrs`.",
			Required:            true,
			CustomType:          widgetConfigType{},
			PlanModifiers:       []planmodifier.String{sameConfigKeepsState{}},
		},
		"layout": schema.SingleNestedAttribute{
			MarkdownDescription: "Position on a 12-column grid. Without it, the widget gets the standard size for its type below the others.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
			Attributes: map[string]schema.Attribute{
				"x": schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(0, 11)}},
				"y": schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.AtLeast(0)}},
				"w": schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(1, 12)}},
				"h": schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(1, 100)}},
			},
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A dashboard over the workspace's telemetry. The resource owns all of the dashboard's widgets: widgets added in Superlog are removed on the next apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The dashboard in Superlog.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 120)}},
			"variables": schema.ListNestedAttribute{
				MarkdownDescription: "Dropdowns shown on the dashboard.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.ObjectType{AttrTypes: variableAttrTypes}, []attr.Value{})),
				Validators:          []validator.List{listvalidator.SizeAtMost(50)},
				NestedObject:        schema.NestedAttributeObject{Attributes: variableAttributes},
			},
			"widgets": schema.ListNestedAttribute{
				MarkdownDescription: "The dashboard's widgets, in order.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.ObjectType{AttrTypes: widgetAttrTypes}, []attr.Value{})),
				Validators:          []validator.List{listvalidator.SizeAtMost(200)},
				NestedObject:        schema.NestedAttributeObject{Attributes: widgetAttributes},
			},
		},
	}
}

var variableAttrTypes = map[string]attr.Type{
	"attribute_key": types.StringType,
	"default_value": types.StringType,
	"label":         types.StringType,
	"multi":         types.BoolType,
	"name":          types.StringType,
	"options":       types.ListType{ElemType: types.StringType},
}

var widgetAttrTypes = map[string]attr.Type{
	"config": widgetConfigType{},
	"id":     types.StringType,
	"layout": types.ObjectType{AttrTypes: layoutTypes},
	"title":  types.StringType,
	"type":   types.StringType,
}

func (r *dashboardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics)
}

func dashboardInput(ctx context.Context, model dashboardModel, diagnostics *diag.Diagnostics) client.DashboardInput {
	input := client.DashboardInput{
		Name:      model.Name.ValueString(),
		Variables: []client.DashboardVariable{},
		Widgets:   []client.Widget{},
	}
	for _, variable := range model.Variables {
		options := variable.Options
		if options == nil {
			options = []string{}
		}
		item := client.DashboardVariable{
			AttributeKey: stringPointer(variable.AttributeKey),
			DefaultValue: stringPointer(variable.DefaultValue),
			Label:        stringPointer(variable.Label),
			Name:         variable.Name.ValueString(),
			Options:      options,
		}
		if !variable.Multi.IsNull() && !variable.Multi.IsUnknown() {
			multi := variable.Multi.ValueBool()
			item.Multi = &multi
		}
		input.Variables = append(input.Variables, item)
	}
	for index, widget := range model.Widgets {
		config := widget.Config.ValueString()
		if !json.Valid([]byte(config)) {
			diagnostics.AddAttributeError(path.Root("widgets").AtListIndex(index).AtName("config"), "Invalid widget config",
				"The config must be a JSON object. Use jsonencode({ ... }).")
			continue
		}
		item := client.Widget{
			Config: json.RawMessage(config),
			Title:  widget.Title.ValueString(),
			Type:   widget.Type.ValueString(),
		}
		if !widget.ID.IsNull() && !widget.ID.IsUnknown() {
			item.ID = widget.ID.ValueString()
		}
		if !widget.Layout.IsNull() && !widget.Layout.IsUnknown() {
			var layout layoutModel
			diagnostics.Append(widget.Layout.As(ctx, &layout, basetypes.ObjectAsOptions{})...)
			item.Layout = &client.WidgetLayout{
				H: layout.H.ValueInt64(),
				W: layout.W.ValueInt64(),
				X: layout.X.ValueInt64(),
				Y: layout.Y.ValueInt64(),
			}
		}
		input.Widgets = append(input.Widgets, item)
	}
	return input
}

// dashboardFromAPI is the dashboard as state. The framework keeps a prior
// config when the server's means the same.
func dashboardFromAPI(dashboard *client.Dashboard, diagnostics *diag.Diagnostics) dashboardModel {
	model := dashboardModel{
		ID:        types.StringValue(dashboard.ID),
		Name:      types.StringValue(dashboard.Name),
		URL:       types.StringValue(dashboard.URL),
		Variables: []variableModel{},
		Widgets:   []widgetModel{},
	}
	for _, variable := range dashboard.Variables {
		item := variableModel{
			AttributeKey: stringValue(variable.AttributeKey),
			DefaultValue: stringValue(variable.DefaultValue),
			Label:        stringValue(variable.Label),
			Multi:        types.BoolNull(),
			Name:         types.StringValue(variable.Name),
			Options:      variable.Options,
		}
		if item.Options == nil {
			item.Options = []string{}
		}
		if variable.Multi != nil {
			item.Multi = types.BoolValue(*variable.Multi)
		}
		model.Variables = append(model.Variables, item)
	}
	for _, widget := range dashboard.Widgets {
		layout := types.ObjectNull(layoutTypes)
		if widget.Layout != nil {
			value, diags := types.ObjectValue(layoutTypes, map[string]attr.Value{
				"h": types.Int64Value(widget.Layout.H),
				"w": types.Int64Value(widget.Layout.W),
				"x": types.Int64Value(widget.Layout.X),
				"y": types.Int64Value(widget.Layout.Y),
			})
			diagnostics.Append(diags...)
			layout = value
		}
		model.Widgets = append(model.Widgets, widgetModel{
			Config: newWidgetConfig(compactJSON(widget.Config)),
			ID:     types.StringValue(widget.ID),
			Layout: layout,
			Title:  types.StringValue(widget.Title),
			Type:   types.StringValue(widget.Type),
		})
	}
	return model
}

func (r *dashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := dashboardInput(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	dashboard, err := r.client.CreateDashboard(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create dashboard", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, appliedDashboard(dashboard, &plan, &resp.Diagnostics))...)
}

// appliedDashboard is the state after a write: the server's dashboard with
// the configs as planned, so a config the server rewrites shows up as a
// change on the next plan instead of failing the apply.
func appliedDashboard(dashboard *client.Dashboard, plan *dashboardModel, diagnostics *diag.Diagnostics) dashboardModel {
	model := dashboardFromAPI(dashboard, diagnostics)
	if len(model.Widgets) != len(plan.Widgets) {
		diagnostics.AddError("Unexpected dashboard", fmt.Sprintf("Superlog saved %d widgets, but %d were sent.", len(model.Widgets), len(plan.Widgets)))
		return model
	}
	for index := range model.Widgets {
		model.Widgets[index].Config = plan.Widgets[index].Config
	}
	return model
}

func (r *dashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dashboard, err := r.client.GetDashboard(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read dashboard", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, dashboardFromAPI(dashboard, &resp.Diagnostics))...)
}

func (r *dashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := dashboardInput(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	dashboard, err := r.client.UpdateDashboard(ctx, state.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update dashboard", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, appliedDashboard(dashboard, &plan, &resp.Diagnostics))...)
}

func (r *dashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDashboard(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete dashboard", err.Error())
	}
}

func (r *dashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
