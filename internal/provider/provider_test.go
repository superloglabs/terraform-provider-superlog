package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Acceptance tests create real objects. Run them with TF_ACC=1 and
// SUPERLOG_API_KEY (and SUPERLOG_ENDPOINT for a local or US workspace) set.
var testProviders = map[string]func() (tfprotov6.ProviderServer, error){
	"superlog": providerserver.NewProtocol6WithError(New("test")()),
}

func testPreCheck(t *testing.T) {
	if os.Getenv("SUPERLOG_API_KEY") == "" {
		t.Fatal("SUPERLOG_API_KEY must be set for acceptance tests")
	}
}
