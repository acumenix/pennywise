package my_hcl

import (
	"fmt"
	"github.com/kaytu-io/pennywise-server/resource"
	"github.com/kaytu-io/pennywise/parser/azurerm"
	usagePackage "github.com/kaytu-io/pennywise/usage"
	"strings"
)

type Resource struct {
	Address string                 `mapstructure:"address"`
	Name    string                 `mapstructure:"name"`
	Type    string                 `mapstructure:"type"`
	Values  map[string]interface{} `mapstructure:"values"`
}

func parseResourcesFromMapStructure(mapStructure map[string]interface{}) (string, []Resource, error) {
	var provider string
	var resources []Resource

	for key, value := range mapStructure {
		labels := strings.Split(key, ".")
		if labels[0] == "provider" {
			provider = labels[1]
		} else if labels[0] == "resource" {
			values, err := value.(map[string]interface{})
			if !err {
				return "", nil, fmt.Errorf("resource %s value is not a map", key)
			}
			resources = append(resources, Resource{
				Address: key,
				Name:    strings.Join(labels[2:], "."),
				Type:    labels[1],
				Values:  values,
			})
		}
	}
	return provider, resources, nil
}

func (r *Resource) ToResourceDef(provider resource.ProviderName, defaultRegion *string) resource.ResourceDef {
	region := ""
	if defaultRegion != nil {
		region = *defaultRegion
	}
	for key, value := range r.Values {
		if provider == resource.AzureProvider && key == "location" {
			region = azurerm.GetRegionCode(value.(string))
			break
		}
	}
	return resource.ResourceDef{
		Address:      r.Address,
		Type:         r.Type,
		Name:         r.Name,
		RegionCode:   region,
		ProviderName: provider,
		Values:       r.Values,
	}
}

func (r *Resource) addUsage(usage usagePackage.Usage) {
	newValues := r.Values

	newValues[usagePackage.Key] = usage.GetUsage(r.Type, r.Address)
	r.Values = newValues
}
