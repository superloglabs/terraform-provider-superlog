# Import a check by its ID, from its URL in Superlog. Secret header values
# are not returned, so the first apply after an import sets them again.
terraform import superlog_synthetic_check.api_health 55555555-5555-4555-8555-555555555555
