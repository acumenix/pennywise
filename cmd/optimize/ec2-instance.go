package optimize

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaytu-io/pennywise/cmd/flags"
	"github.com/kaytu-io/pennywise/cmd/optimize/view"
	"github.com/kaytu-io/pennywise/pkg/api/wastage"
	"github.com/kaytu-io/pennywise/pkg/server"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"sync"
	"time"
)

var ec2InstanceCommand = &cobra.Command{
	Use:   "ec2-instance",
	Short: ``,
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile := flags.ReadStringFlag(cmd, "profile")
		config, err := server.GetConfig()
		if err != nil {
			return err
		}

		ctx := context.Background()

		cfg, err := GetConfig(ctx, "", "", "", "", &profile, nil)
		if err != nil {
			return err
		}
		cfg.Region = "us-east-2"
		regionClient := ec2.NewFromConfig(cfg)

		regions, err := regionClient.DescribeRegions(ctx, &ec2.DescribeRegionsInput{AllRegions: aws.Bool(false)})
		if err != nil {
			return err
		}

		m := view.NewEC2InstanceList()
		wg := sync.WaitGroup{}
		wg.Add(len(regions.Regions))

		for _, region := range regions.Regions {
			localCfg := cfg
			localCfg.Region = *region.RegionName

			go func() {
				client := ec2.NewFromConfig(localCfg)
				paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})
				for paginator.HasMorePages() {
					page, err := paginator.NextPage(ctx)
					if err != nil {
						panic(err)
					}

					for _, r := range page.Reservations {
						for _, v := range r.Instances {
							if v.State.Name != types.InstanceStateNameRunning {
								continue
							}

							m.SendItem(view.Item{
								Instance: v,
								Region:   localCfg.Region,
							})
						}
					}
				}
				wg.Done()
			}()
		}

		go func() {
			wg.Wait()
			m.Finished()
		}()

		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			return err
		}

		if selectedItem := m.SelectedItem(); selectedItem != nil {
			cfg.Region = selectedItem.Region

			m := view.NewProgressBar(
				fmt.Sprintf("Getting possible optimizations for: %s", *selectedItem.Instance.InstanceId))

			var goErr error
			var res *wastage.EC2InstanceWastageResponse
			go func() {
				m.UpdateProgressBar(0.10)
				defer m.UpdateProgressBar(1)

				req, err := getEc2InstanceRequestData(ctx, cfg, selectedItem.Instance, selectedItem.Region)
				if err != nil {
					goErr = err
					return
				}
				m.UpdateProgressBar(0.50)
				res, goErr = wastage.Ec2InstanceWastageRequest(*req, config.AccessToken)
			}()

			p := tea.NewProgram(m)
			if _, err := p.Run(); err != nil {
				return err
			}

			if goErr != nil {
				return goErr
			}

			fmt.Println("InstanceID:", *selectedItem.Instance.InstanceId)
			fmt.Println("CurrentCost:", res.CurrentCost)
			fmt.Println("TotalSavings:", res.TotalSavings)
			fmt.Println("Recommendations:")
			fmt.Println("----------------------------------------------------------------------------")
			for _, recom := range res.Recommendations {
				fmt.Printf("%s; Total saving: %v\n", recom.Description, recom.Saving)
			}
			fmt.Println("----------------------------------------------------------------------------")
		}

		return nil
	},
}

func getEc2InstanceRequestData(ctx context.Context, cfg aws.Config, instance types.Instance, region string) (*wastage.EC2InstanceWastageRequest, error) {
	client := ec2.NewFromConfig(cfg)

	var volumes []types.Volume
	for _, bd := range instance.BlockDeviceMappings {
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
		Region:   region,
	}, nil
}
