package azurerm

import (
	azurermtf "github.com/kaytu.io/pennywise/server/azurerm/terraform"
	"github.com/kaytu.io/pennywise/server/terraform"
)

// RegistryName is the fully qualified name under which this provider is stored in the registry.
const RegistryName = "registry.terraform.io/hashicorp/azurerm"

// TerraformProviderInitializer is a terraform.ProviderInitializer that initializes the default GCP provider.
var TerraformProviderInitializer = terraform.ProviderInitializer{
	MatchNames: []string{ProviderName, RegistryName},
	Provider: func(values map[string]interface{}) (terraform.Provider, error) {
		return azurermtf.NewProvider(ProviderName)
	},
}
