terraform {
  required_providers {
    superlog = {
      source = "superloglabs/superlog"
    }
  }
}

# Set SUPERLOG_API_KEY to an API key from Settings → API keys.
provider "superlog" {
  # https://superlog.sh for the EU region, https://us.superlog.sh for the US region.
  endpoint = "https://superlog.sh"
}
