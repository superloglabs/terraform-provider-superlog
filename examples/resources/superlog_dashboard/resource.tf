resource "superlog_dashboard" "checkout" {
  name = "Checkout"

  variables = [
    {
      name          = "env"
      label         = "Environment"
      attribute_key = "deployment.environment"
      options       = ["production", "staging"]
      default_value = "production"
    },
  ]

  widgets = [
    {
      title = "Errors by service"
      type  = "timeseries_count"
      # $${env} is the `env` variable; $$ keeps Terraform from reading it.
      config = jsonencode({
        source  = "logs"
        groupBy = "service.name"
        filter = {
          resourceAttrs = [{ key = "deployment.environment", value = "$${env}" }]
        }
      })
      layout = { x = 0, y = 0, w = 6, h = 4 }
    },
    {
      title  = "Request duration p95"
      type   = "timeseries_metric"
      config = jsonencode({ metricName = "http.server.duration", aggregation = "p95", unit = "duration_ms" })
      layout = { x = 6, y = 0, w = 6, h = 4 }
    },
    {
      title  = "Recent errors"
      type   = "log_table"
      config = jsonencode({ source = "logs", filter = { resourceAttrs = [{ key = "severity", value = "ERROR" }] } })
      layout = { x = 0, y = 4, w = 12, h = 6 }
    },
  ]
}
