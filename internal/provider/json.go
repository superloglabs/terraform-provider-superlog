package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// sameWidgetConfig reports whether two widget configs mean the same thing.
// Key order and spacing do not matter, and the server fills in an empty
// filter, so empty objects and null values count as left out.
func sameWidgetConfig(a, b string) bool {
	var left, right any
	if json.Unmarshal([]byte(a), &left) != nil || json.Unmarshal([]byte(b), &right) != nil {
		return a == b
	}
	return reflect.DeepEqual(withoutEmpty(left), withoutEmpty(right))
}

func withoutEmpty(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := map[string]any{}
		for key, item := range typed {
			cleaned := withoutEmpty(item)
			if cleaned == nil {
				continue
			}
			if object, ok := cleaned.(map[string]any); ok && len(object) == 0 {
				continue
			}
			result[key] = cleaned
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for i, item := range typed {
			result[i] = withoutEmpty(item)
		}
		return result
	default:
		return value
	}
}

// compactJSON is the server's config as a JSON string with sorted keys,
// without the empty objects and nulls the server fills in.
func compactJSON(raw json.RawMessage) string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return string(raw)
	}
	encoded, err := json.Marshal(withoutEmpty(value))
	if err != nil {
		return string(raw)
	}
	return string(encoded)
}

// widgetConfigType is a JSON string whose values are equal when
// sameWidgetConfig says so, so equivalent configs plan no change.
type widgetConfigType struct {
	basetypes.StringType
}

var (
	_ basetypes.StringTypable                    = widgetConfigType{}
	_ basetypes.StringValuableWithSemanticEquals = widgetConfig{}
)

func (t widgetConfigType) Equal(o attr.Type) bool {
	_, ok := o.(widgetConfigType)
	return ok
}

func (t widgetConfigType) String() string { return "widgetConfigType" }

func (t widgetConfigType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return widgetConfig{StringValue: in}, nil
}

func (t widgetConfigType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	value, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := value.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", value)
	}
	return widgetConfig{StringValue: stringValue}, nil
}

func (t widgetConfigType) ValueType(_ context.Context) attr.Value { return widgetConfig{} }

type widgetConfig struct {
	basetypes.StringValue
}

func (v widgetConfig) Equal(o attr.Value) bool {
	other, ok := o.(widgetConfig)
	return ok && v.StringValue.Equal(other.StringValue)
}

func (v widgetConfig) Type(_ context.Context) attr.Type { return widgetConfigType{} }

func (v widgetConfig) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	other, ok := newValuable.(widgetConfig)
	if !ok {
		return false, nil
	}
	return sameWidgetConfig(v.ValueString(), other.ValueString()), nil
}

func newWidgetConfig(value string) widgetConfig {
	return widgetConfig{StringValue: basetypes.NewStringValue(value)}
}

// sameConfigKeepsState plans no change when the configured widget config
// means the same as the one in state, such as after an import.
type sameConfigKeepsState struct{}

func (m sameConfigKeepsState) Description(_ context.Context) string {
	return "Keeps the config in state when the configured JSON means the same."
}

func (m sameConfigKeepsState) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m sameConfigKeepsState) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if sameWidgetConfig(req.StateValue.ValueString(), req.ConfigValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
