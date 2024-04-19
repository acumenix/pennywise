package view

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	preferences2 "github.com/kaytu-io/pennywise/cmd/optimize/preferences"
	"github.com/kaytu-io/pennywise/pkg/api/wastage"
	"github.com/kaytu-io/pennywise/pkg/hash"
	"github.com/kaytu-io/pennywise/pkg/server"
	"github.com/muesli/reflow/wordwrap"
	"golang.org/x/net/context"
	"strings"
	"sync"
	"time"
)

type App struct {
	status              string
	statusErr           string
	errorChan           chan error
	statusChan          chan string
	processInstanceChan chan OptimizationItem

	optimizationsTable *Ec2InstanceOptimizations

	counter      int64
	counterMutex sync.RWMutex
	width        int
	height       int
}

var (
	helpStyle  = list.DefaultStyles().HelpStyle.PaddingLeft(4).Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func NewApp(cfg aws.Config, accountHash string) *App {
	pi := make(chan OptimizationItem, 1000)
	r := &App{
		status:              "",
		errorChan:           make(chan error, 1000),
		statusChan:          make(chan string, 1000),
		processInstanceChan: pi,
		optimizationsTable:  NewEC2InstanceOptimizations(pi),
	}
	go r.UpdateStatus()
	go r.ProcessAllRegions(cfg)
	go r.ProcessInstances(cfg, accountHash)
	return r
}

func (m *App) Init() tea.Cmd {
	optTableCmd := m.optimizationsTable.Init()

	return tea.Batch(optTableCmd, tea.EnterAltScreen)
}

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
	sb := strings.Builder{}
	sb.WriteString(m.optimizationsTable.View())
	sb.WriteString("\n  status: " + m.status + "\n")
	if len(m.statusErr) > 0 {
		sb.WriteString(errorStyle.Render(wordwrap.String("  error: "+m.statusErr, m.width)) + "\n")
	}
	sb.WriteString("\n\n")
	return sb.String()
}

func (m *App) UpdateStatus() {
	for {
		select {
		case err := <-m.errorChan:
			m.statusErr = fmt.Sprintf("Failed due to %v", err)
		case newStatus := <-m.statusChan:
			m.status = newStatus
		}
	}
}

func (m *App) ProcessInstances(awsCfg aws.Config, accountHash string) {
	config, err := server.GetConfig()
	if err != nil {
		m.errorChan <- err
		return
	}

	for item := range m.processInstanceChan {
		awsCfg.Region = item.Region
		localAWSCfg := awsCfg
		localItem := item
		m.counterMutex.Lock()
		m.counter++
		localCounter := m.counter
		m.counterMutex.Unlock()
		m.statusChan <- fmt.Sprintf("calculating possible optimizations for %d instances.", localCounter)

		go func() {
			m.ProcessInstance(config, localAWSCfg, localItem, accountHash)
			m.counterMutex.Lock()
			defer m.counterMutex.Unlock()
			m.counter--
			m.statusChan <- fmt.Sprintf("calculating possible optimizations for %d instances.", m.counter)
		}()
	}
}

func (m *App) ProcessInstance(config *server.Config, awsConf aws.Config, item OptimizationItem, accountHash string) {
	defer func() {
		if r := recover(); r != nil {
			m.errorChan <- fmt.Errorf("%v", r)
		}
	}()

	req, err := getEc2InstanceRequestData(context.Background(), awsConf, item.Instance, preferences2.Export(item.Preferences), accountHash)
	if err != nil {
		m.errorChan <- err
		return
	}
	res, err := wastage.Ec2InstanceWastageRequest(*req, config.AccessToken)
	if err != nil {
		m.errorChan <- err
		return
	}
	if res.RightSizing == nil {
		item.OptimizationLoading = false
		m.optimizationsTable.SendItem(item)
		return
	}

	m.optimizationsTable.SendItem(OptimizationItem{
		Instance:                  item.Instance,
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
		Preferences:               item.Preferences,
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
				if v.InstanceLifecycle == types.InstanceLifecycleTypeSpot {
					continue
				}
				isAutoScaling := false
				for _, tag := range v.Tags {
					if *tag.Key == "aws:autoscaling:groupName" && tag.Value != nil && *tag.Value != "" {
						isAutoScaling = true
					}
				}
				if isAutoScaling {
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
	m.statusChan <- "Retrieving data from AWS"
	defer func() {
		if r := recover(); r != nil {
			m.errorChan <- fmt.Errorf("%v", r)
			return
		}

		m.statusChan <- "Successfully fetched all ec2 instances from AWS. Calculating instance optimizations..."
	}()

	m.statusChan <- "Listing all available regions"
	regionClient := ec2.NewFromConfig(cfg)
	regions, err := regionClient.DescribeRegions(context.Background(), &ec2.DescribeRegionsInput{AllRegions: aws.Bool(false)})
	if err != nil {
		m.errorChan <- err
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(regions.Regions))

	m.statusChan <- "Fetching all EC2 Instances"
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

func getEc2InstanceRequestData(ctx context.Context, cfg aws.Config, instance types.Instance, preferences map[string]*string, accountHash string) (*wastage.EC2InstanceWastageRequest, error) {
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
	startTime := time.Now().Add(-24 * 7 * time.Hour)
	endTime := time.Now()
	statistics := []types2.Statistic{
		types2.StatisticAverage,
		types2.StatisticMinimum,
		types2.StatisticMaximum,
	}
	dimensionFilter := []types2.Dimension{
		{
			Name:  aws.String("InstanceId"),
			Value: instance.InstanceId,
		},
	}
	metrics := map[string][]types2.Datapoint{}

	paginator := cloudwatch.NewListMetricsPaginator(cloudwatchClient, &cloudwatch.ListMetricsInput{
		Namespace: aws.String("AWS/EC2"),
		Dimensions: []types2.DimensionFilter{
			{
				Name:  aws.String("InstanceId"),
				Value: instance.InstanceId,
			},
		},
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range page.Metrics {
			if p.MetricName == nil || (*p.MetricName != "CPUUtilization" &&
				*p.MetricName != "NetworkIn" &&
				*p.MetricName != "NetworkOut") {
				continue
			}

			// Create input for GetMetricStatistics
			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("AWS/EC2"),
				MetricName: p.MetricName,
				Dimensions: dimensionFilter,
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

	paginator = cloudwatch.NewListMetricsPaginator(cloudwatchClient, &cloudwatch.ListMetricsInput{
		Namespace: aws.String("CWAgent"),
		Dimensions: []types2.DimensionFilter{
			{
				Name:  aws.String("InstanceId"),
				Value: instance.InstanceId,
			},
		},
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range page.Metrics {
			if p.MetricName == nil || (*p.MetricName != "mem_used_percent") {
				continue
			}

			// Create input for GetMetricStatistics
			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("CWAgent"),
				MetricName: p.MetricName,
				Dimensions: dimensionFilter,
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

	var monitoring *types.MonitoringState
	if instance.Monitoring != nil {
		monitoring = &instance.Monitoring.State
	}
	var placement *wastage.EC2Placement
	if instance.Placement != nil {
		placement = &wastage.EC2Placement{
			Tenancy:          instance.Placement.Tenancy,
			AvailabilityZone: *instance.Placement.AvailabilityZone,
			HashedHostId:     hash.HashString(*instance.Placement.HostId),
		}
	}

	var volumes []wastage.EC2Volume
	for _, v := range res.Volumes {
		volumes = append(volumes, toEBSVolume(v))
	}

	return &wastage.EC2InstanceWastageRequest{
		HashedAccountID: accountHash,
		Instance: wastage.EC2Instance{
			HashedInstanceId:  hash.HashString(*instance.InstanceId),
			State:             instance.State.Name,
			InstanceType:      instance.InstanceType,
			Platform:          instance.Platform,
			ThreadsPerCore:    *instance.CpuOptions.ThreadsPerCore,
			CoreCount:         *instance.CpuOptions.CoreCount,
			EbsOptimized:      *instance.EbsOptimized,
			InstanceLifecycle: instance.InstanceLifecycle,
			Monitoring:        monitoring,
			Placement:         placement,
		},
		Volumes:     volumes,
		Metrics:     metrics,
		Region:      cfg.Region,
		Preferences: preferences,
	}, nil
}

func toEBSVolume(v types.Volume) wastage.EC2Volume {
	return wastage.EC2Volume{
		HashedVolumeId: hash.HashString(*v.VolumeId),
		VolumeType:     v.VolumeType,
		Size:           *v.Size,
		Iops:           *v.Iops,
	}
}
