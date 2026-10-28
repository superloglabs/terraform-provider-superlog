data "superlog_slack_channel" "alerts" {
  name = "alerts"
}

# More than 50 error logs from checkout in production in 5 minutes.
resource "superlog_alert" "checkout_errors" {
  name        = "Checkout errors"
  source      = "logs"
  aggregation = "count"
  comparator  = "gt"
  threshold   = 50
  filter = {
    service  = "checkout"
    severity = "ERROR"
    attributes = [
      { key = "deployment.environment", value = "production" },
    ]
  }
  slack = {
    channel_id             = data.superlog_slack_channel.alerts.channel_id
    integration_account_id = data.superlog_slack_channel.alerts.integration_account_id
  }
}

# p95 span duration above 750 ms, checked for each service.
resource "superlog_alert" "slow_services" {
  name           = "Slow services"
  source         = "traces"
  aggregation    = "p95"
  field          = "duration"
  comparator     = "gt"
  threshold      = 750
  window_minutes = 10
  group_mode     = "per_group"
  group_by       = "service.name"
}
