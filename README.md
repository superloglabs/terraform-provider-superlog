# Terraform provider for Superlog

Manage [Superlog](https://superlog.sh) alerts, dashboards, and synthetic checks with Terraform or OpenTofu.

```hcl
terraform {
  required_providers {
    superlog = {
      source = "superloglabs/superlog"
    }
  }
}

provider "superlog" {
  # https://superlog.sh (EU, the default) or https://us.superlog.sh (US).
  endpoint = "https://superlog.sh"
}

resource "superlog_alert" "checkout_errors" {
  name        = "Checkout errors"
  source      = "logs"
  aggregation = "count"
  comparator  = "gt"
  threshold   = 50
  filter      = { service = "checkout", severity = "ERROR" }
}
```

Set `SUPERLOG_API_KEY` to an API key from **Settings → API keys** in Superlog. See the [documentation](docs/index.md) for every resource.

| Resource | What it manages |
| --- | --- |
| `superlog_alert` | Threshold alerts on logs, spans, or metrics |
| `superlog_dashboard` | Dashboards, their variables, and all of their widgets |
| `superlog_synthetic_check` | Scheduled HTTP checks |
| `superlog_slack_channel` (data source) | Looks up a Slack channel by name for an alert or check |

## Exporting what you already have

Superlog writes Terraform for existing objects, with an `import` block for each one, so `terraform apply` adopts them instead of creating copies:

```shell
curl -s -H "Authorization: Bearer $SUPERLOG_API_KEY" https://superlog.sh/api/v1/terraform | jq -r .hcl > superlog.tf
terraform init && terraform apply
```

**Copy as Terraform** on an alert, dashboard, or synthetic check page copies the same configuration for one object.

## Development

```shell
go test ./...                 # unit tests
go generate ./...             # regenerate docs/ (needs terraform on PATH)
```

Acceptance tests create real objects in the workspace of the API key:

```shell
TF_ACC=1 SUPERLOG_API_KEY=slk_... SUPERLOG_ENDPOINT=https://superlog.sh go test ./internal/provider -run TestAcc
```

To try a local build, point Terraform at it with `dev_overrides` in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/superloglabs/superlog" = "/path/to/go/bin"
  }
  direct {}
}
```

## Releasing

Push a tag such as `v0.1.0`. The release workflow builds and signs the binaries with goreleaser, and the Terraform Registry publishes the new version.
