package terraform

import (
	"github.com/kaytu-io/pennywise-server/client"
	"github.com/kaytu-io/pennywise-server/schema"
	"github.com/kaytu-io/pennywise/parser/aws"
	"github.com/kaytu-io/pennywise/parser/azurerm"
	"github.com/kaytu-io/pennywise/parser/terraform"
	"github.com/kaytu-io/pennywise/usage"
	"io"
	"os"
)

var (
	ServerClientAddress = os.Getenv("SERVER_CLIENT_URL")
)

// EstimateTerraformPlanJson is a helper function that reads a Terraform plan json file using the provided io.Reader,
// calculates the costs of the resources and show them.
// It uses the Backend to retrieve the pricing data.
func EstimateTerraformPlanJson(plan io.Reader, u usage.Usage) error {
	providerInitializers := []terraform.ProviderInitializer{
		aws.TerraformProviderInitializer,
		azurerm.TerraformProviderInitializer,
	}

	tfplan := terraform.NewPlan(providerInitializers...)
	if err := tfplan.Read(plan); err != nil {
		return err
	}
	tfplan.SetUsage(u)

	plannedQueries, err := tfplan.ExtractPlannedQueries()
	if err != nil {
		return err
	}
	var resources []schema.ResourceDef
	for _, rs := range plannedQueries {
		res := rs.ToResource("")
		resources = append(resources, res)
	}
	serverClient := client.NewPennywiseServerClient(ServerClientAddress)
	state := schema.State{Resources: resources}
	_, err = serverClient.GetStateCost(state)
	if err != nil {
		return err
	}
	return nil
}
