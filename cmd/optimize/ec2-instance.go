package optimize

import (
	"github.com/aws/aws-sdk-go-v2/service/sts"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaytu-io/pennywise/cmd/flags"
	"github.com/kaytu-io/pennywise/cmd/optimize/view"
	awsConfig "github.com/kaytu-io/pennywise/pkg/aws"
	"github.com/kaytu-io/pennywise/pkg/hash"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var ec2InstanceCommand = &cobra.Command{
	Use:   "ec2-instance",
	Short: ``,
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile := flags.ReadStringFlag(cmd, "profile")
		cfg, err := awsConfig.GetConfig(context.Background(), "", "", "", "", &profile, nil)
		if err != nil {
			return err
		}

		client := sts.NewFromConfig(cfg)
		out, err := client.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		accountHash := hash.HashString(*out.Account)
		userIdHash := hash.HashString(*out.UserId)
		arnHash := hash.HashString(*out.Arn)

		p := tea.NewProgram(view.NewApp(cfg, accountHash, userIdHash, arnHash))
		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}
