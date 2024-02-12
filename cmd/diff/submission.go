package diff

import (
	"github.com/kaytu-io/pennywise/cmd/flags"
	outputDiff "github.com/kaytu-io/pennywise/pkg/output/diff"
	"github.com/kaytu-io/pennywise/pkg/schema"
	"github.com/kaytu-io/pennywise/pkg/server"
	"github.com/spf13/cobra"
)

var submissionCommand = &cobra.Command{
	Use:   "submission",
	Short: `Shows a submission cost.`,
	Long:  `Shows a submission cost.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		classic := flags.ReadBooleanFlag(cmd, "classic")

		submissionId := flags.ReadStringFlag(cmd, "submission-id")
		compareTo := flags.ReadStringFlag(cmd, "compare-to")

		err := submissionsDiff(classic, submissionId, compareTo, DefaultServerAddress)
		if err != nil {
			return err
		}

		return nil
	},
}

func submissionsDiff(classic bool, submissionId, compareToId string, ServerClientAddress string) error {
	serverClient, err := server.NewPennywiseServerClient(ServerClientAddress)
	if err != nil {
		return err
	}
	sub, err := schema.ReadSubmissionFile(submissionId)
	if err != nil {
		return err
	}
	compareTo, err := schema.ReadSubmissionFile(compareToId)
	if err != nil {
		return err
	}

	req := schema.SubmissionsDiff{
		Current:   *sub,
		CompareTo: *compareTo,
	}
	stateDiff, err := serverClient.GetSubmissionsDiff(req)
	if err != nil {
		return err
	}
	//if classic {
	//	costString, err := state.CostString()
	//	if err != nil {
	//		return err
	//	}
	//	fmt.Println(costString)
	//	fmt.Println("To learn how to use usage open:\nhttps://github.com/kaytu-io/pennywise/blob/main/docs/usage.md")
	//} else {
	//	err = output.ShowStateCosts(state)
	//	if err != nil {
	//		return err
	//	}
	//}
	err = outputDiff.ShowStateCosts(stateDiff)
	if err != nil {
		return err
	}
	return nil
}
