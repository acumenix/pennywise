package optimize

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaytu-io/pennywise/cmd/flags"
	"github.com/kaytu-io/pennywise/cmd/optimize/view"
	awsConfig "github.com/kaytu-io/pennywise/pkg/aws"
	"github.com/kaytu-io/pennywise/pkg/server"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var ec2InstanceCommand = &cobra.Command{
	Use:   "ec2-instance",
	Short: ``,
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile := flags.ReadStringFlag(cmd, "profile")
		_, err := server.GetConfig()
		if err != nil {
			return err
		}

		cfg, err := awsConfig.GetConfig(context.Background(), "", "", "", "", &profile, nil)
		if err != nil {
			return err
		}

		p := tea.NewProgram(view.NewApp(cfg))
		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}
