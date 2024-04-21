package wastage

import (
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type AWSCredential struct {
	AccountID string `json:"accountID"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type EC2Placement struct {
	Tenancy          types.Tenancy `json:"tenancy"`
	AvailabilityZone string        `json:"availabilityZone"`
	HashedHostId     string        `json:"hashedHostId"`
}

type EC2Instance struct {
	HashedInstanceId  string                      `json:"hashedInstanceId"`
	State             types.InstanceStateName     `json:"state"`
	InstanceType      types.InstanceType          `json:"instanceType"`
	Platform          types.PlatformValues        `json:"platform"`
	ThreadsPerCore    int32                       `json:"threadsPerCore"`
	CoreCount         int32                       `json:"coreCount"`
	EbsOptimized      bool                        `json:"ebsOptimized"`
	InstanceLifecycle types.InstanceLifecycleType `json:"instanceLifecycle"`
	Monitoring        *types.MonitoringState      `json:"monitoring"`
	Placement         *EC2Placement               `json:"placement"`
}

type EC2Volume struct {
	HashedVolumeId string           `json:"hashedVolumeId"`
	VolumeType     types.VolumeType `json:"volumeType"`
	Size           int32            `json:"size"`
	Iops           int32            `json:"iops"`
}

type EC2InstanceWastageRequest struct {
	Instance    EC2Instance                   `json:"instance"`
	Volumes     []EC2Volume                   `json:"volumes"`
	Metrics     map[string][]types2.Datapoint `json:"metrics"`
	Region      string                        `json:"region"`
	Preferences map[string]*string            `json:"preferences"`
}

type RightSizingRecommendation struct {
	TargetInstanceType string  `json:"targetInstanceType"`
	Saving             float64 `json:"saving"`
	CurrentCost        float64 `json:"currentCost"`
	TargetCost         float64 `json:"targetCost"`

	AvgCPUUsage string `json:"avgCPUUsage"`
	TargetCores string `json:"targetCores"`

	AvgNetworkBandwidth       string `json:"avgNetworkBandwidth"`
	TargetNetworkPerformance  string `json:"targetNetworkBandwidth"`
	CurrentNetworkPerformance string `json:"currentNetworkPerformance"`

	CurrentMemory string `json:"currentMemory"`
	TargetMemory  string `json:"targetMemory"`
}

type EC2InstanceWastageResponse struct {
	CurrentCost  float64                    `json:"currentCost"`
	TotalSavings float64                    `json:"totalSavings"`
	RightSizing  *RightSizingRecommendation `json:"rightSizing"`
}
