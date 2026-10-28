package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlert(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	config := func(threshold int) string {
		return fmt.Sprintf(`
resource "superlog_alert" "test" {
  name        = %q
  source      = "logs"
  aggregation = "count"
  comparator  = "gt"
  threshold   = %d
  filter = {
    service    = "checkout"
    attributes = [{ key = "deployment.environment", op = "neq", value = "staging" }]
  }
}`, name, threshold)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testPreCheck(t) },
		ProtoV6ProviderFactories: testProviders,
		Steps: []resource.TestStep{
			{
				Config: config(50),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superlog_alert.test", "threshold", "50"),
					resource.TestCheckResourceAttr("superlog_alert.test", "window_minutes", "5"),
					resource.TestCheckResourceAttrSet("superlog_alert.test", "url"),
				),
			},
			{Config: config(80), Check: resource.TestCheckResourceAttr("superlog_alert.test", "threshold", "80")},
			{ResourceName: "superlog_alert.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestAccDashboard(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	errors := `{
      title  = "Errors"
      type   = "timeseries_count"
      config = jsonencode({ source = "logs", groupBy = "service.name", filter = { resourceAttrs = [{ key = "deployment.environment", value = "$${env}" }] } })
      layout = { x = 0, y = 0, w = 6, h = 4 }
    }`
	notes := `{
      title  = "Notes"
      type   = "markdown"
      config = jsonencode({ markdown = "Runbook" })
      layout = { x = 6, y = 0, w = 6, h = 4 }
    }`
	config := func(widgets string) string {
		return fmt.Sprintf(`
resource "superlog_dashboard" "test" {
  name      = %q
  variables = [{ name = "env", options = ["prod", "staging"], default_value = "prod" }]
  widgets   = [%s]
}`, name, widgets)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testPreCheck(t) },
		ProtoV6ProviderFactories: testProviders,
		Steps: []resource.TestStep{
			{
				Config: config(errors + "," + notes),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superlog_dashboard.test", "widgets.#", "2"),
					resource.TestCheckResourceAttr("superlog_dashboard.test", "widgets.1.title", "Notes"),
				),
			},
			{
				Config: config(notes),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superlog_dashboard.test", "widgets.#", "1"),
					resource.TestCheckResourceAttr("superlog_dashboard.test", "widgets.0.title", "Notes"),
				),
			},
			{ResourceName: "superlog_dashboard.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestAccSyntheticCheck(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	config := func(token string) string {
		return fmt.Sprintf(`
resource "superlog_synthetic_check" "test" {
  name             = %q
  url              = "https://example.com/health"
  interval_seconds = 900
  headers          = { Accept = "application/json" }
  secret_headers   = { Authorization = %q }
  assertions = [
    { kind = "status_code", op = "eq", value = "2xx" },
    { kind = "response_time", op = "lt", value = "1500", severity = "degraded" },
  ]
}`, name, token)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testPreCheck(t) },
		ProtoV6ProviderFactories: testProviders,
		Steps: []resource.TestStep{
			{
				Config: config("Bearer one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superlog_synthetic_check.test", "method", "GET"),
					resource.TestCheckResourceAttr("superlog_synthetic_check.test", "assertions.0.severity", "critical"),
					resource.TestCheckResourceAttr("superlog_synthetic_check.test", "secret_headers.Authorization", "Bearer one"),
				),
			},
			{Config: config("Bearer two"), Check: resource.TestCheckResourceAttr("superlog_synthetic_check.test", "secret_headers.Authorization", "Bearer two")},
			{
				ResourceName:      "superlog_synthetic_check.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Superlog never returns secret header values.
				ImportStateVerifyIgnore: []string{"secret_headers.Authorization"},
			},
		},
	})
}
