package resource

import (
	"github.com/kaytu-io/pennywise/server/internal/price"
	"github.com/kaytu-io/pennywise/server/internal/product"
	"github.com/shopspring/decimal"
)

// Resource is a single resource definition.
type ResourceDef struct {
	Address      string                 `json:"address"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	RegionCode   string                 `json:"region_code"`
	ProviderName ProviderName           `json:"provider_name"`
	Values       map[string]interface{} `json:"values"`
}

type State struct {
	Resources []ResourceDef `json:"resources"`
}

// Resource represents a single cloud resource. It has a unique Address and a collection of multiple
// Component queries.
type Resource struct {
	// Address uniquely identifies this cloud Resource.
	Address string

	// Provider is the cloud provider that this Resource belongs to.
	Provider ProviderName

	// Type describes the type of the Resource.
	Type string

	// Components is a list of price components that make up this Resource. If it is empty, the resource
	// is considered to be skipped.
	Components []Component
}

// Component represents a price component of a cloud Resource. It is used to fetch the price for a single
// component of a resource. For example, a compute instance might be have different pricing for the number
// of CPU's, amount of RAM, etc. - each of these would be a Component.
type Component struct {
	Name            string
	HourlyQuantity  decimal.Decimal
	MonthlyQuantity decimal.Decimal
	Unit            string
	Details         []string
	Usage           bool
	ProductFilter   *product.Filter
	PriceFilter     *price.Filter
}
