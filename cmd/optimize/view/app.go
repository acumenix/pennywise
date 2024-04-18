package view

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	tea "github.com/charmbracelet/bubbletea"
	preferences2 "github.com/kaytu-io/pennywise/cmd/optimize/preferences"
	"github.com/kaytu-io/pennywise/pkg/api/wastage"
	"github.com/kaytu-io/pennywise/pkg/server"
	"golang.org/x/net/context"
	"sync"
	"time"
)

type App struct {
	status              string
	errorChan           chan error
	processInstanceChan chan OptimizationItem

	optimizationsTable *Ec2InstanceOptimizations
}

func NewApp(cfg aws.Config) *App {
	pi := make(chan OptimizationItem, 1000)
	r := &App{
		status:              "",
		errorChan:           make(chan error, 1000),
		processInstanceChan: pi,
		optimizationsTable:  NewEC2InstanceOptimizations(pi),
	}
	go r.ProcessAllRegions(cfg)
	go r.ProcessInstances(cfg)
	return r
}

func (m *App) Init() tea.Cmd {
	optTableCmd := m.optimizationsTable.Init()

	return tea.Batch(optTableCmd, tea.EnterAltScreen)
}

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	_, optTableCmd := m.optimizationsTable.Update(msg)
	return m, tea.Batch(optTableCmd)
}

func (m *App) View() string {
	return "\n  Status: " + m.status + "..." + "\n\n" +
		m.optimizationsTable.View()
}

func (m *App) ProcessInstances(awsCfg aws.Config) {
	config, err := server.GetConfig()
	if err != nil {
		m.errorChan <- err
		return
	}

	for item := range m.processInstanceChan {
		awsCfg.Region = item.Region
		go m.ProcessInstance(config, awsCfg, item.Instance, item.Preferences)
	}
}

func (m *App) ProcessInstance(config *server.Config, awsConf aws.Config, instance types.Instance, preferences []preferences2.PreferenceItem) {
	defer func() {
		if r := recover(); r != nil {
			m.errorChan <- fmt.Errorf("%v", r)
		}
	}()

	req, err := getEc2InstanceRequestData(context.Background(), awsConf, instance, preferences2.Export(preferences))
	if err != nil {
		m.errorChan <- err
		return
	}
	res, err := wastage.Ec2InstanceWastageRequest(*req, config.AccessToken)
	if err != nil {
		m.errorChan <- err
		return
	}

	m.optimizationsTable.SendItem(OptimizationItem{
		Instance:                  instance,
		Region:                    awsConf.Region,
		OptimizationLoading:       false,
		TargetInstanceType:        res.RightSizing.TargetInstanceType,
		TotalSaving:               res.RightSizing.Saving,
		CurrentCost:               res.RightSizing.CurrentCost,
		TargetCost:                res.RightSizing.TargetCost,
		AvgCPUUsage:               res.RightSizing.AvgCPUUsage,
		TargetCores:               res.RightSizing.TargetCores,
		AvgNetworkBandwidth:       res.RightSizing.AvgNetworkBandwidth,
		TargetNetworkPerformance:  res.RightSizing.TargetNetworkPerformance,
		CurrentNetworkPerformance: res.RightSizing.CurrentNetworkPerformance,
		CurrentMemory:             res.RightSizing.CurrentMemory,
		TargetMemory:              res.RightSizing.TargetMemory,
		Preferences:               preferences,
	})
}

func (m *App) ProcessRegion(cfg aws.Config) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			m.errorChan <- fmt.Errorf("%v", r)
		}
	}()
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			m.errorChan <- err
			return
		}

		for _, r := range page.Reservations {
			for _, v := range r.Instances {
				if v.State.Name != types.InstanceStateNameRunning {
					continue
				}

				preferences := preferences2.DefaultPreferences()
				oi := OptimizationItem{
					Instance:            v,
					Region:              cfg.Region,
					OptimizationLoading: true,
					TargetInstanceType:  "",
					TotalSaving:         0,
					Preferences:         preferences,
				}
				m.optimizationsTable.SendItem(oi)
				m.processInstanceChan <- oi
			}
		}
	}
}

func (m *App) ProcessAllRegions(cfg aws.Config) {
	m.status = "Retrieving data from AWS"
	defer func() {
		if r := recover(); r != nil {
			m.status = fmt.Sprintf("Failed to retrieve data! Panic: %v", r)
			return
		}

		var err error
		select {
		case err = <-m.errorChan:
			m.status = fmt.Sprintf("Failed to retrieve data! Error: %v", err)
			return
		default:
		}

		m.status = "Successfully finished loading all the data"
	}()

	m.status = "Listing all available regions"
	regionClient := ec2.NewFromConfig(cfg)
	regions, err := regionClient.DescribeRegions(context.Background(), &ec2.DescribeRegionsInput{AllRegions: aws.Bool(false)})
	if err != nil {
		m.errorChan <- err
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(regions.Regions))

	m.status = "Fetching all EC2 Instances"
	for _, region := range regions.Regions {
		localCfg := cfg
		localCfg.Region = *region.RegionName

		go func() {
			defer wg.Done()
			m.ProcessRegion(localCfg)
		}()
	}
	wg.Wait()
}

func getEc2InstanceRequestData(ctx context.Context, cfg aws.Config, instance types.Instance, preferences map[string]*string) (*wastage.EC2InstanceWastageRequest, error) {
	client := ec2.NewFromConfig(cfg)

	var volumeIds []string
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs == nil {
			continue
		}
		volumeIds = append(volumeIds, *bd.Ebs.VolumeId)
	}

	res, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: volumeIds,
	})
	if err != nil {
		return nil, err
	}

	cloudwatchClient := cloudwatch.NewFromConfig(cfg)
	paginator := cloudwatch.NewListMetricsPaginator(cloudwatchClient, &cloudwatch.ListMetricsInput{
		Namespace: aws.String("AWS/EC2"),
		Dimensions: []types2.DimensionFilter{
			{
				Name:  aws.String("InstanceId"),
				Value: instance.InstanceId,
			},
		},
	})
	startTime := time.Now().Add(-24 * 7 * time.Hour)
	endTime := time.Now()

	metrics := map[string][]types2.Datapoint{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range page.Metrics {
			statistics := []types2.Statistic{
				types2.StatisticAverage,
				types2.StatisticMinimum,
				types2.StatisticMaximum,
			}

			// Create input for GetMetricStatistics
			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("AWS/EC2"),
				MetricName: p.MetricName,
				Dimensions: []types2.Dimension{
					{
						Name:  aws.String("InstanceId"),
						Value: instance.InstanceId,
					},
				},
				StartTime:  aws.Time(startTime),
				EndTime:    aws.Time(endTime),
				Period:     aws.Int32(60 * 60), // 1 hour intervals
				Statistics: statistics,
			}

			// Get metric data
			resp, err := cloudwatchClient.GetMetricStatistics(ctx, input)
			if err != nil {
				return nil, err
			}

			metrics[*p.MetricName] = resp.Datapoints
		}
	}
	return &wastage.EC2InstanceWastageRequest{
		Instance:    instance,
		Volumes:     res.Volumes,
		Metrics:     metrics,
		Region:      cfg.Region,
		Preferences: preferences,
	}, nil
}
