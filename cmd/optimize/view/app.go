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

	client := ec2.NewFromConfig(awsConf)
	var volumeIds []string
	for _, bd := range item.Instance.BlockDeviceMappings {
		if bd.Ebs == nil {
			continue
		}
		volumeIds = append(volumeIds, *bd.Ebs.VolumeId)
	}
	volumesResp, err := client.DescribeVolumes(context.Background(), &ec2.DescribeVolumesInput{
		VolumeIds: volumeIds,
	})
	if err != nil {
		m.errorChan <- err
		return
	}

	req, err := getEc2InstanceRequestData(context.Background(), awsConf, item.Instance, volumesResp.Volumes, preferences2.Export(item.Preferences), accountHash)
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
		Volumes:                   volumesResp.Volumes,
		Region:                    awsConf.Region,
		OptimizationLoading:       false,
		RightSizingRecommendation: *res.RightSizing,
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

func getEc2InstanceRequestData(ctx context.Context, cfg aws.Config, instance types.Instance, volumes []types.Volume, preferences map[string]*string, accountHash string) (*wastage.EC2InstanceWastageRequest, error) {

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
			Tenancy: instance.Placement.Tenancy,
		}
		if instance.Placement.AvailabilityZone != nil {
			placement.AvailabilityZone = *instance.Placement.AvailabilityZone
		}
		if instance.Placement.HostId != nil {
			placement.HashedHostId = hash.HashString(*instance.Placement.HostId)
		}
	}

	var kaytuVolumes []wastage.EC2Volume
	volumeMetrics := map[string]map[string][]types2.Datapoint{}
	for _, v := range volumes {
		kaytuVolumes = append(kaytuVolumes, toEBSVolume(v))

		paginator := cloudwatch.NewListMetricsPaginator(cloudwatchClient, &cloudwatch.ListMetricsInput{
			Namespace: aws.String("AWS/EBS"),
			Dimensions: []types2.DimensionFilter{
				{
					Name:  aws.String("VolumeId"),
					Value: v.VolumeId,
				},
			},
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, p := range page.Metrics {
				if p.MetricName == nil || (*p.MetricName != "VolumeReadOps" &&
					*p.MetricName != "VolumeWriteOps" &&
					*p.MetricName != "VolumeReadBytes" &&
					*p.MetricName != "VolumeWriteBytes") {
					continue
				}

				// Create input for GetMetricStatistics
				input := &cloudwatch.GetMetricStatisticsInput{
					Namespace:  aws.String("AWS/EBS"),
					MetricName: p.MetricName,
					Dimensions: []types2.Dimension{
						{
							Name:  aws.String("VolumeId"),
							Value: v.VolumeId,
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

				if _, ok := volumeMetrics[hash.HashString(*v.VolumeId)]; !ok {
					volumeMetrics[hash.HashString(*v.VolumeId)] = make(map[string][]types2.Datapoint)
				}
				volumeMetrics[hash.HashString(*v.VolumeId)][*p.MetricName] = resp.Datapoints
			}
		}
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
		Volumes:       kaytuVolumes,
		Metrics:       metrics,
		VolumeMetrics: volumeMetrics,
		Region:        cfg.Region,
		Preferences:   preferences,
	}, nil
}

func toEBSVolume(v types.Volume) wastage.EC2Volume {
	return wastage.EC2Volume{
		HashedVolumeId:   hash.HashString(*v.VolumeId),
		VolumeType:       v.VolumeType,
		Size:             v.Size,
		Iops:             v.Iops,
		AvailabilityZone: v.AvailabilityZone,
		Throughput:       v.Throughput,
	}
}
