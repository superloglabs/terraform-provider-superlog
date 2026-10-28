package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorsIncludeIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer slk_test" {
			t.Errorf("missing bearer token")
		}
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":   "invalid_request",
			"error":  "Invalid widget",
			"issues": []map[string]any{{"message": "Choose a metric", "path": []any{"widgets", 1, "config", "metricName"}}},
		})
	}))
	defer server.Close()
	c, err := New(server.URL, "slk_test", "test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateDashboard(context.Background(), DashboardInput{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "widgets.1.config.metricName: Choose a metric") {
		t.Fatalf("unexpected error: %v", err)
	}
	if IsNotFound(err) {
		t.Fatal("a 400 is not a not-found")
	}
}

func TestNotFoundNeedsAnObjectCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		code := "alert_not_found"
		if strings.HasPrefix(r.URL.Path, "/api/v1/dashboards") {
			// The feature is off: the object may still exist.
			code = "dashboards_unavailable"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "error": "x"})
	}))
	defer server.Close()
	c, _ := New(server.URL, "slk_test", "test")
	if _, err := c.GetAlert(context.Background(), "a"); !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := c.GetDashboard(context.Background(), "d"); IsNotFound(err) {
		t.Fatal("an unavailable feature must not read as a deleted dashboard")
	}
}

func TestEndpointMustBeAURL(t *testing.T) {
	if _, err := New("superlog.sh", "k", "test"); err == nil {
		t.Fatal("expected an error for an endpoint without a scheme")
	}
	c, err := New("", "k", "test")
	if err != nil || c.baseURL != "https://superlog.sh/api/v1" {
		t.Fatalf("default endpoint: %v %v", c, err)
	}
}
