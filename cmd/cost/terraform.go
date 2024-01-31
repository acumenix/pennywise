package cost

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/pennywise/cmd/cost/terraform"
	"github.com/kaytu-io/pennywise/cmd/flags"
	"github.com/kaytu-io/pennywise/pkg/output"
	"github.com/kaytu-io/pennywise/pkg/parser/hcl"
	"github.com/kaytu-io/pennywise/pkg/schema"
	"github.com/kaytu-io/pennywise/pkg/server"
	"github.com/kaytu-io/pennywise/pkg/submission"
	usagePackage "github.com/kaytu-io/pennywise/pkg/usage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

var terraformCommand = &cobra.Command{
	Use:   "terraform",
	Short: `Shows the costs by parsing terraform resources.`,
	Long:  `Shows the costs by parsing terraform resources.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		usagePath := flags.ReadStringOptionalFlag(cmd, "usage")
		var usage usagePackage.Usage
		if usagePath != nil {
			usageFile, err := os.Open(*usagePath)
			if err != nil {
				return fmt.Errorf("error while reading usage file %s", err)
			}
			defer usageFile.Close()

			ext := filepath.Ext(*usagePath)
			switch ext {
			case ".json":
				err = json.NewDecoder(usageFile).Decode(&usage)
			case ".yaml", ".yml":
				err = yaml.NewDecoder(usageFile).Decode(&usage)
			default:
				return fmt.Errorf("unsupported file format %s for usage file", ext)
			}
			if err != nil {
				return fmt.Errorf("error while parsing usage file %s", err)
			}

		} else {
			usage = usagePackage.Usage{}
		}

		ServerClientAddress := flags.ReadStringFlag(cmd, "server-url")
		if os.Getenv("SERVER_CLIENT_URL") != "" {
			ServerClientAddress = os.Getenv("SERVER_CLIENT_URL")
		}

		classic := flags.ReadBooleanFlag(cmd, "classic")

		jsonPath := flags.ReadStringOptionalFlag(cmd, "json-path")
		err := estimateTfPlanJson(classic, *jsonPath, usage, ServerClientAddress)
		if err != nil {
			return err
		}
		return nil
	},
}

func estimateTfProject(classic bool, projectDir string, usage usagePackage.Usage, ServerClientAddress string) error {
	provider, hclResources, err := hcl.ParseHclResources(projectDir, usage)
	if err != nil {
		return err
	}
	var resources []schema.ResourceDef
	for _, res := range hclResources {
		resources = append(resources, res.ToResourceDef(provider, nil))
	}
	sub, err := submission.CreateSubmission(resources)
	if err != nil {
		return err
	}
	err = sub.StoreAsFile()
	if err != nil {
		return err
	}
	serverClient, err := server.NewPennywiseServerClient(ServerClientAddress)
	if err != nil {
		return err
	}
	state, err := serverClient.GetStateCost(*sub)
	if err != nil {
		return err
	}
	if classic {
		costString, err := state.CostString()
		if err != nil {
			return err
		}
		fmt.Println(costString)
	} else {
		err = output.ShowStateCosts(state)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func estimateTfPlanJson(classic bool, jsonPath string, usage usagePackage.Usage, ServerClientAddress string) error {
	file, err := os.Open(jsonPath)
	if err != nil {
		return err
	}
	resources, err := terraform.ParseTerraformPlanJson(file, usage)
	if err != nil {
		return err
	}
	serverClient, err := server.NewPennywiseServerClient(ServerClientAddress)
	if err != nil {
		return err
	}
	sub, err := submission.CreateSubmission(resources)
	if err != nil {
		return err
	}
	err = sub.StoreAsFile()
	if err != nil {
		return err
	}
	state, err := serverClient.GetStateCost(*sub)
	if err != nil {
		return err
	}
	if classic {
		costString, err := state.CostString()
		if err != nil {
			return err
		}
		fmt.Println(costString)
	} else {
		err = output.ShowStateCosts(state)
		if err != nil {
			return err
		}
	}
	return nil
}
