package client

import (
	"context"
	"encoding/json"
	"net/http"
)

type SlackTarget struct {
	ChannelID            string `json:"channelId"`
	IntegrationAccountID string `json:"integrationAccountId"`
}

type AttributeFilter struct {
	Key   string  `json:"key"`
	Op    *string `json:"op,omitempty"`
	Value string  `json:"value"`
}

type AlertFilter struct {
	MetricKind    *string           `json:"metricKind,omitempty"`
	MetricUnit    *string           `json:"metricUnit,omitempty"`
	MinDurationMs *float64          `json:"minDurationMs,omitempty"`
	ResourceAttrs []AttributeFilter `json:"resourceAttrs,omitempty"`
	Service       *string           `json:"service,omitempty"`
	Severity      *string           `json:"severity,omitempty"`
	SpanName      *string           `json:"spanName,omitempty"`
	StatusCode    *string           `json:"statusCode,omitempty"`
}

// AlertInput is every field of an alert a client sets.
type AlertInput struct {
	Aggregation               string       `json:"aggregation"`
	Comparator                string       `json:"comparator"`
	Enabled                   bool         `json:"enabled"`
	EvaluationIntervalSeconds int64        `json:"evaluationIntervalSeconds"`
	Field                     *string      `json:"field"`
	Filter                    AlertFilter  `json:"filter"`
	GroupBy                   *string      `json:"groupBy"`
	GroupMode                 string       `json:"groupMode"`
	MetricName                *string      `json:"metricName"`
	Name                      string       `json:"name"`
	Slack                     *SlackTarget `json:"slack"`
	Source                    string       `json:"source"`
	Threshold                 float64      `json:"threshold"`
	WindowMinutes             int64        `json:"windowMinutes"`
}

type Alert struct {
	AlertInput
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (c *Client) ListAlerts(ctx context.Context) ([]Alert, error) {
	var out struct {
		Alerts []Alert `json:"alerts"`
	}
	err := c.do(ctx, http.MethodGet, "/alerts", nil, &out)
	return out.Alerts, err
}

func (c *Client) GetAlert(ctx context.Context, id string) (*Alert, error) {
	var out struct {
		Alert Alert `json:"alert"`
	}
	if err := c.do(ctx, http.MethodGet, "/alerts/"+escape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Alert, nil
}

func (c *Client) CreateAlert(ctx context.Context, input AlertInput) (*Alert, error) {
	var out struct {
		Alert Alert `json:"alert"`
	}
	if err := c.do(ctx, http.MethodPost, "/alerts", input, &out); err != nil {
		return nil, err
	}
	return &out.Alert, nil
}

func (c *Client) UpdateAlert(ctx context.Context, id string, input AlertInput) (*Alert, error) {
	var out struct {
		Alert Alert `json:"alert"`
	}
	if err := c.do(ctx, http.MethodPatch, "/alerts/"+escape(id), input, &out); err != nil {
		return nil, err
	}
	return &out.Alert, nil
}

func (c *Client) DeleteAlert(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/alerts/"+escape(id), nil, nil)
}

type DashboardVariable struct {
	AttributeKey *string  `json:"attributeKey,omitempty"`
	DefaultValue *string  `json:"defaultValue,omitempty"`
	Label        *string  `json:"label,omitempty"`
	Multi        *bool    `json:"multi,omitempty"`
	Name         string   `json:"name"`
	Options      []string `json:"options"`
}

type WidgetLayout struct {
	H int64 `json:"h"`
	W int64 `json:"w"`
	X int64 `json:"x"`
	Y int64 `json:"y"`
}

type Widget struct {
	Config json.RawMessage `json:"config"`
	ID     string          `json:"id,omitempty"`
	Layout *WidgetLayout   `json:"layout,omitempty"`
	Title  string          `json:"title"`
	Type   string          `json:"type"`
}

type DashboardInput struct {
	Name      string              `json:"name"`
	Variables []DashboardVariable `json:"variables"`
	Widgets   []Widget            `json:"widgets"`
}

type Dashboard struct {
	DashboardInput
	ID  string `json:"id"`
	URL string `json:"url"`
}

type DashboardSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListDashboards(ctx context.Context) ([]DashboardSummary, error) {
	var out struct {
		Dashboards []DashboardSummary `json:"dashboards"`
	}
	err := c.do(ctx, http.MethodGet, "/dashboards", nil, &out)
	return out.Dashboards, err
}

func (c *Client) GetDashboard(ctx context.Context, id string) (*Dashboard, error) {
	var out struct {
		Dashboard Dashboard `json:"dashboard"`
	}
	if err := c.do(ctx, http.MethodGet, "/dashboards/"+escape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Dashboard, nil
}

func (c *Client) CreateDashboard(ctx context.Context, input DashboardInput) (*Dashboard, error) {
	var out struct {
		Dashboard Dashboard `json:"dashboard"`
	}
	if err := c.do(ctx, http.MethodPost, "/dashboards", input, &out); err != nil {
		return nil, err
	}
	return &out.Dashboard, nil
}

// UpdateDashboard replaces the dashboard's name, variables, and widgets.
func (c *Client) UpdateDashboard(ctx context.Context, id string, input DashboardInput) (*Dashboard, error) {
	var out struct {
		Dashboard Dashboard `json:"dashboard"`
	}
	if err := c.do(ctx, http.MethodPatch, "/dashboards/"+escape(id), input, &out); err != nil {
		return nil, err
	}
	return &out.Dashboard, nil
}

func (c *Client) DeleteDashboard(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/dashboards/"+escape(id), nil, nil)
}

type SyntheticAssertion struct {
	Key      *string `json:"key,omitempty"`
	Kind     string  `json:"kind"`
	Op       string  `json:"op"`
	Severity string  `json:"severity"`
	Value    string  `json:"value"`
}

type SyntheticHeader struct {
	Name   string  `json:"name"`
	Secret bool    `json:"secret"`
	Value  *string `json:"value"`
}

type SyntheticCheckInput struct {
	Assertions      []SyntheticAssertion `json:"assertions"`
	Body            *string              `json:"body"`
	Enabled         bool                 `json:"enabled"`
	FailuresToAlert int64                `json:"failuresToAlert"`
	FollowRedirects bool                 `json:"followRedirects"`
	Headers         []SyntheticHeader    `json:"headers"`
	IntervalSeconds int64                `json:"intervalSeconds"`
	Method          string               `json:"method"`
	Name            string               `json:"name"`
	Retries         int64                `json:"retries"`
	Slack           *SlackTarget         `json:"slack"`
	TimeoutMs       int64                `json:"timeoutMs"`
	URL             string               `json:"url"`
}

type SyntheticCheck struct {
	SyntheticCheckInput
	ID     string `json:"id"`
	WebURL string `json:"webUrl"`
}

func (c *Client) ListSyntheticChecks(ctx context.Context) ([]SyntheticCheck, error) {
	var out struct {
		Checks []SyntheticCheck `json:"checks"`
	}
	err := c.do(ctx, http.MethodGet, "/synthetic-checks", nil, &out)
	return out.Checks, err
}

func (c *Client) GetSyntheticCheck(ctx context.Context, id string) (*SyntheticCheck, error) {
	var out struct {
		Check SyntheticCheck `json:"check"`
	}
	if err := c.do(ctx, http.MethodGet, "/synthetic-checks/"+escape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Check, nil
}

func (c *Client) CreateSyntheticCheck(ctx context.Context, input SyntheticCheckInput) (*SyntheticCheck, error) {
	var out struct {
		Check SyntheticCheck `json:"check"`
	}
	if err := c.do(ctx, http.MethodPost, "/synthetic-checks", input, &out); err != nil {
		return nil, err
	}
	return &out.Check, nil
}

func (c *Client) UpdateSyntheticCheck(ctx context.Context, id string, input SyntheticCheckInput) (*SyntheticCheck, error) {
	var out struct {
		Check SyntheticCheck `json:"check"`
	}
	if err := c.do(ctx, http.MethodPatch, "/synthetic-checks/"+escape(id), input, &out); err != nil {
		return nil, err
	}
	return &out.Check, nil
}

func (c *Client) DeleteSyntheticCheck(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/synthetic-checks/"+escape(id), nil, nil)
}

type SlackChannel struct {
	AccountName          string `json:"accountName"`
	ChannelID            string `json:"channelId"`
	IntegrationAccountID string `json:"integrationAccountId"`
	Name                 string `json:"name"`
	Private              bool   `json:"private"`
}

func (c *Client) ListSlackChannels(ctx context.Context) ([]SlackChannel, error) {
	var out struct {
		Channels []SlackChannel `json:"channels"`
	}
	err := c.do(ctx, http.MethodGet, "/slack-channels", nil, &out)
	return out.Channels, err
}
