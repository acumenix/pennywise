package resources

import (
	"github.com/kaytu-io/pennywise/server/internal/query"
	"github.com/mitchellh/mapstructure"
)

type PrivateDNSSRVRecord struct {
	provider *Provider

	location string
	// Usage
	monthlyQueries *int64
}

type privateDNSSRVRecordValues struct {
	ResourceGroupName ResourceGroupNameStruct `mapstructure:"resource_group_name"`

	Usage struct {
		MonthlyQueries int64 `mapstructure:"monthly_queries"`
	} `mapstructure:"tc_usage"`
}

func (p *Provider) newPrivateDNSSRVRecord(vals privateDNSSRVRecordValues) *PrivateDNSSRVRecord {
	inst := &PrivateDNSSRVRecord{
		provider: p,

		location:       vals.ResourceGroupName.Values.Location,
		monthlyQueries: &vals.Usage.MonthlyQueries,
	}
	return inst
}

func decoderPrivateDnsSRVRecord(tfVals map[string]interface{}) (privateDNSSRVRecordValues, error) {
	var v privateDNSSRVRecordValues
	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &v,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return v, err
	}

	if err := decoder.Decode(tfVals); err != nil {
		return v, err
	}
	return v, nil
}

func (inst *PrivateDNSSRVRecord) component() []query.Component {
	region := getLocationName(inst.location)
	return privateDNSSRVRecord(inst.provider.key, region, inst.monthlyQueries)
}

func privateDNSSRVRecord(key, region string, monthlyQueries *int64) []query.Component {
	return DNSQueriesCostComponent(key, region, monthlyQueries)
}
