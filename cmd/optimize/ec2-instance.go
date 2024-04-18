package optimize

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaytu-io/pennywise/cmd/flags"
	"github.com/kaytu-io/pennywise/cmd/optimize/view"
	"github.com/spf13/cobra"
)

var ec2InstanceCommand = &cobra.Command{
	Use:   "ec2-instance",
	Short: ``,
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile := flags.ReadStringFlag(cmd, "profile")

		p := tea.NewProgram(view.NewApp(profile))
		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}
