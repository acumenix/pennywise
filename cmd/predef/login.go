package predef

import (
	"fmt"
	"github.com/kaytu-io/pennywise/pkg/api/auth0"
	"github.com/kaytu-io/pennywise/pkg/server"
	"github.com/spf13/cobra"
	"time"
)

const RetrySleep = 3
const DefaultWorkspace = "kaytu"

var LoginCmd = &cobra.Command{
	Use: "login",
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceCode, err := auth0.RequestDeviceCode()
		if err != nil {
			return fmt.Errorf("[login-deviceCode]: %v", err)
		}

		var accessToken string
		for i := 0; i < 100; i++ {
			accessToken, err = auth0.AccessToken(deviceCode)
			if err != nil {
				time.Sleep(RetrySleep * time.Second)
				continue
			}
			break
		}
		if err != nil {
			return fmt.Errorf("[login-accessToken]: %v", err)
		}

		err = server.SetConfig(server.Config{
			AccessToken:      accessToken,
			DefaultWorkspace: DefaultWorkspace,
		})
		if err != nil {
			return fmt.Errorf("[login-setConfig]: %v", err)
		}
		return nil
	},
}
