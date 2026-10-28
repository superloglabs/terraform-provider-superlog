package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/superloglabs/terraform-provider-superlog/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = &syntheticCheckResource{}
	_ resource.ResourceWithImportState = &syntheticCheckResource{}
)

type syntheticCheckResource struct {
	client *client.Client
}

type assertionModel struct {
	Key      types.String `tfsdk:"key"`
	Kind     types.String `tfsdk:"kind"`
	Op       types.String `tfsdk:"op"`
	Severity types.String `tfsdk:"severity"`
	Value    types.String `tfsdk:"value"`
}

type syntheticCheckModel struct {
	Assertions      []assertionModel  `tfsdk:"assertions"`
	Body            types.String      `tfsdk:"body"`
	Enabled         types.Bool        `tfsdk:"enabled"`
	FailuresToAlert types.Int64       `tfsdk:"failures_to_alert"`
	FollowRedirects types.Bool        `tfsdk:"follow_redirects"`
	Headers         map[string]string `tfsdk:"headers"`
	ID              types.String      `tfsdk:"id"`
	IntervalSeconds types.Int64       `tfsdk:"interval_seconds"`
	Method          types.String      `tfsdk:"method"`
	Name            types.String      `tfsdk:"name"`
	Retries         types.Int64       `tfsdk:"retries"`
	SecretHeaders   map[string]string `tfsdk:"secret_headers"`
	Slack           *slackModel       `tfsdk:"slack"`
	TimeoutMs       types.Int64       `tfsdk:"timeout_ms"`
	URL             types.String      `tfsdk:"url"`
	WebURL          types.String      `tfsdk:"web_url"`
}

var assertionAttrTypes = map[string]attr.Type{
	"key":      types.StringType,
	"kind":     types.StringType,
	"op":       types.StringType,
	"severity": types.StringType,
	"value":    types.StringType,
}

func newSyntheticCheckResource() resource.Resource { return &syntheticCheckResource{} }

func (r *syntheticCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_check"
}

func (r *syntheticCheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	emptyMap := mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{}))
	resp.Schema = schema.Schema{
		MarkdownDescription: "An HTTP check that requests a URL on a schedule and checks the response. After `failures_to_alert` failed runs in a row it posts to its Slack channel, and posts the recovery in the same thread.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"web_url": schema.StringAttribute{
				MarkdownDescription: "The check in Superlog.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 200)}},
			"url":  schema.StringAttribute{Required: true, MarkdownDescription: "The full HTTP or HTTPS URL to request."},
			"method": schema.StringAttribute{
				MarkdownDescription: "Defaults to `GET`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("GET"),
				Validators:          []validator.String{stringvalidator.OneOf("GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS")},
			},
			"body": schema.StringAttribute{Optional: true, MarkdownDescription: "The request body. Not for `GET` or `HEAD`."},
			"headers": schema.MapAttribute{
				MarkdownDescription: "Request headers by name.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             emptyMap,
			},
			"secret_headers": schema.MapAttribute{
				MarkdownDescription: "Request headers stored encrypted, such as `Authorization`. Superlog never returns their values, so changes made outside Terraform are not detected.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Default:             emptyMap,
			},
			"interval_seconds": schema.Int64Attribute{
				MarkdownDescription: "How often the check runs: 60, 300, 900, 1800, or 3600 seconds. Defaults to 300.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(300),
				Validators:          []validator.Int64{int64validator.OneOf(60, 300, 900, 1800, 3600)},
			},
			"timeout_ms": schema.Int64Attribute{
				MarkdownDescription: "From 1000 to 30000. Defaults to 10000.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(10000),
				Validators:          []validator.Int64{int64validator.Between(1000, 30000)},
			},
			"retries": schema.Int64Attribute{
				MarkdownDescription: "Immediate retries before a run counts as failed, from 0 to 2. Defaults to 1.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				Validators:          []validator.Int64{int64validator.Between(0, 2)},
			},
			"failures_to_alert": schema.Int64Attribute{
				MarkdownDescription: "Failed runs in a row before the check posts to Slack, from 1 to 10. Defaults to 1.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				Validators:          []validator.Int64{int64validator.Between(1, 10)},
			},
			"follow_redirects": schema.BoolAttribute{
				MarkdownDescription: "Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "A disabled check does not run. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"assertions": schema.ListNestedAttribute{
				MarkdownDescription: "Checks on the response. Without any, a run passes when the request completes.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.ObjectType{AttrTypes: assertionAttrTypes}, []attr.Value{})),
				Validators:          []validator.List{listvalidator.SizeAtMost(20)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kind": schema.StringAttribute{
							MarkdownDescription: "`status_code`, `response_time` (ms), `header`, `body`, `json_path`, or `certificate_expiry` (days).",
							Required:            true,
							Validators: []validator.String{stringvalidator.OneOf(
								"status_code", "response_time", "header", "body", "json_path", "certificate_expiry",
							)},
						},
						"op": schema.StringAttribute{
							MarkdownDescription: "`status_code` takes `eq`, `neq`, `lt`, `gt`; `response_time` `lt`; `header` `eq`, `neq`, `contains`, `not_contains`, `exists`, `not_exists`; `body` `contains`, `not_contains`; `json_path` all of these; `certificate_expiry` `gt`.",
							Required:            true,
							Validators: []validator.String{stringvalidator.OneOf(
								"eq", "neq", "lt", "gt", "contains", "not_contains", "exists", "not_exists",
							)},
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "The header name for `header`, or the JSON path for `json_path`, such as `$.status`.",
							Optional:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Compared with the response. A status code may be a class such as `2xx`.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString(""),
						},
						"severity": schema.StringAttribute{
							MarkdownDescription: "A failed `critical` assertion fails the run; a `degraded` one marks it degraded. Defaults to `critical`.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("critical"),
							Validators:          []validator.String{stringvalidator.OneOf("critical", "degraded")},
						},
					},
				},
			},
			"slack": slackAttribute("check"),
		},
	}
}

func (r *syntheticCheckResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics)
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func syntheticCheckInput(model syntheticCheckModel) client.SyntheticCheckInput {
	input := client.SyntheticCheckInput{
		Assertions:      []client.SyntheticAssertion{},
		Body:            stringPointer(model.Body),
		Enabled:         model.Enabled.ValueBool(),
		FailuresToAlert: model.FailuresToAlert.ValueInt64(),
		FollowRedirects: model.FollowRedirects.ValueBool(),
		Headers:         []client.SyntheticHeader{},
		IntervalSeconds: model.IntervalSeconds.ValueInt64(),
		Method:          model.Method.ValueString(),
		Name:            model.Name.ValueString(),
		Retries:         model.Retries.ValueInt64(),
		Slack:           slackTarget(model.Slack),
		TimeoutMs:       model.TimeoutMs.ValueInt64(),
		URL:             model.URL.ValueString(),
	}
	for _, name := range sortedKeys(model.Headers) {
		value := model.Headers[name]
		input.Headers = append(input.Headers, client.SyntheticHeader{Name: name, Value: &value})
	}
	for _, name := range sortedKeys(model.SecretHeaders) {
		value := model.SecretHeaders[name]
		input.Headers = append(input.Headers, client.SyntheticHeader{Name: name, Secret: true, Value: &value})
	}
	for _, assertion := range model.Assertions {
		input.Assertions = append(input.Assertions, client.SyntheticAssertion{
			Key:      stringPointer(assertion.Key),
			Kind:     assertion.Kind.ValueString(),
			Op:       assertion.Op.ValueString(),
			Severity: assertion.Severity.ValueString(),
			Value:    assertion.Value.ValueString(),
		})
	}
	return input
}

// syntheticCheckFromAPI is the check as state. Secret header values come
// from prior, since the API does not return them.
func syntheticCheckFromAPI(check *client.SyntheticCheck, prior map[string]string) syntheticCheckModel {
	model := syntheticCheckModel{
		Assertions:      []assertionModel{},
		Body:            stringValue(check.Body),
		Enabled:         types.BoolValue(check.Enabled),
		FailuresToAlert: types.Int64Value(check.FailuresToAlert),
		FollowRedirects: types.BoolValue(check.FollowRedirects),
		Headers:         map[string]string{},
		ID:              types.StringValue(check.ID),
		IntervalSeconds: types.Int64Value(check.IntervalSeconds),
		Method:          types.StringValue(check.Method),
		Name:            types.StringValue(check.Name),
		Retries:         types.Int64Value(check.Retries),
		SecretHeaders:   map[string]string{},
		Slack:           slackFromAPI(check.Slack),
		TimeoutMs:       types.Int64Value(check.TimeoutMs),
		URL:             types.StringValue(check.URL),
		WebURL:          types.StringValue(check.WebURL),
	}
	for _, header := range check.Headers {
		if header.Secret {
			model.SecretHeaders[header.Name] = prior[header.Name]
		} else if header.Value != nil {
			model.Headers[header.Name] = *header.Value
		} else {
			model.Headers[header.Name] = ""
		}
	}
	for _, assertion := range check.Assertions {
		key := types.StringNull()
		if assertion.Key != nil && *assertion.Key != "" {
			key = types.StringValue(*assertion.Key)
		}
		model.Assertions = append(model.Assertions, assertionModel{
			Key:      key,
			Kind:     types.StringValue(assertion.Kind),
			Op:       types.StringValue(assertion.Op),
			Severity: types.StringValue(assertion.Severity),
			Value:    types.StringValue(assertion.Value),
		})
	}
	return model
}

func (r *syntheticCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan syntheticCheckModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := r.client.CreateSyntheticCheck(ctx, syntheticCheckInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create synthetic check", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, syntheticCheckFromAPI(check, plan.SecretHeaders))...)
}

func (r *syntheticCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state syntheticCheckModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := r.client.GetSyntheticCheck(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read synthetic check", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, syntheticCheckFromAPI(check, state.SecretHeaders))...)
}

func (r *syntheticCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state syntheticCheckModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := r.client.UpdateSyntheticCheck(ctx, state.ID.ValueString(), syntheticCheckInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to update synthetic check", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, syntheticCheckFromAPI(check, plan.SecretHeaders))...)
}

func (r *syntheticCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state syntheticCheckModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSyntheticCheck(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete synthetic check", err.Error())
	}
}

func (r *syntheticCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
