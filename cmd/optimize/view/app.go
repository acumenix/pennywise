package view

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaytu-io/pennywise/pkg/api/wastage"
	awsConfig "github.com/kaytu-io/pennywise/pkg/aws"
	"github.com/kaytu-io/pennywise/pkg/server"
	"golang.org/x/net/context"
	"sync"
	"time"
)

type App struct {
	Profile   string
	status    string
	errorChan chan error

	optimizationsTable *Ec2InstanceOptimizations
}

func NewApp(profile string) *App {
	r := &App{
		Profile:            profile,
		status:             "",
		errorChan:          make(chan error, 1000),
		optimizationsTable: NewEC2InstanceOptimizations(),
	}
	go r.StartProcess(profile)
	return r
}

func (m *App) Init() tea.Cmd {
	optTableCmd := m.optimizationsTable.Init()

	return tea.Batch(optTableCmd)
}

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	_, optTableCmd := m.optimizationsTable.Update(msg)
	return m, tea.Batch(optTableCmd)
}

func (m *App) View() string {
	return "\n  Status: " + m.status + "\n\n" +
		m.optimizationsTable.View()
}

func (m *App) StartProcess(profile string) {
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

	ctx := context.Background()
	config, err := server.GetConfig()
	if err != nil {
		m.errorChan <- err
		return
	}

	m.status = "Authenticating"
	cfg, err := awsConfig.GetConfig(ctx, "", "", "", "", &profile, nil)
	if err != nil {
		m.errorChan <- err
		return
	}

	m.status = "Listing all available regions"
	regionClient := ec2.NewFromConfig(cfg)
	regions, err := regionClient.DescribeRegions(ctx, &ec2.DescribeRegionsInput{AllRegions: aws.Bool(false)})
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
			wgOptimizations := sync.WaitGroup{}
			defer func() {
				if r := recover(); r != nil {
					m.errorChan <- err
				}

				wgOptimizations.Wait()
				wg.Done()
			}()
			client := ec2.NewFromConfig(localCfg)
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

						m.optimizationsTable.SendItem(OptimizationItem{
							Instance:            v,
							Region:              localCfg.Region,
							OptimizationLoading: true,
							TargetInstanceType:  "",
							TotalSaving:         0,
						})

						awsConf := localCfg
						localInstance := v
						wgOptimizations.Add(1)
						go func() {
							defer func() {
								if r := recover(); r != nil {
									m.errorChan <- err
								}
								wgOptimizations.Done()
							}()

							req, err := getEc2InstanceRequestData(context.Background(), awsConf, localInstance)
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
								Instance:            localInstance,
								Region:              localCfg.Region,
								OptimizationLoading: false,
								TargetInstanceType:  res.RightSizing.TargetInstanceType,
								TotalSaving:         res.RightSizing.Saving,
							})
						}()
					}
				}
			}
		}()
	}

	wg.Wait()
	m.optimizationsTable.Finished()
}

func getEc2InstanceRequestData(ctx context.Context, cfg aws.Config, instance types.Instance) (*wastage.EC2InstanceWastageRequest, error) {
	client := ec2.NewFromConfig(cfg)

	var volumes []types.Volume
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs == nil {
			continue
		}
		res, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			VolumeIds: []string{*bd.Ebs.VolumeId},
		})
		if err != nil {
			return nil, err
		}

		if len(res.Volumes) == 0 {
			return nil, fmt.Errorf("volume not found")
		}
		volumes = append(volumes, res.Volumes...)
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
		Instance: instance,
		Metrics:  metrics,
		Volumes:  volumes,
		Region:   cfg.Region,
	}, nil
}
