package optimize

import "github.com/spf13/cobra"

// OptimizeCmd diff commands
var OptimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: `Get optimization recommendations for every cloud resource you're using.`,
	Long:  `Get optimization recommendations for every cloud resource you're using.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	OptimizeCmd.AddCommand(ec2InstanceCommand)
	ec2InstanceCommand.Flags().String("profile", "", "AWS profile for authentication")

}
