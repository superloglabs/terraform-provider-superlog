variable "health_token" {
  type      = string
  sensitive = true
}

resource "superlog_synthetic_check" "api_health" {
  name              = "API health"
  url               = "https://api.example.com/health"
  interval_seconds  = 60
  failures_to_alert = 2

  headers        = { Accept = "application/json" }
  secret_headers = { Authorization = "Bearer ${var.health_token}" }

  assertions = [
    { kind = "status_code", op = "eq", value = "2xx" },
    { kind = "json_path", key = "$.status", op = "eq", value = "ok" },
    { kind = "response_time", op = "lt", value = "1500", severity = "degraded" },
    { kind = "certificate_expiry", op = "gt", value = "14", severity = "degraded" },
  ]
}
