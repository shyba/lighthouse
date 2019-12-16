package cmd

import (
	"github.com/lbryio/lighthouse/app"
	"github.com/lbryio/lighthouse/app/actions"
	"github.com/lbryio/lighthouse/app/config"
	"github.com/lbryio/lighthouse/app/jobs"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	serveCmd.PersistentFlags().StringP("host", "", "0.0.0.0", "host to listen on")
	serveCmd.PersistentFlags().IntP("port", "p", 50005, "port binding used for the api server")
	//Bind to Viper
	viper.BindPFlag("host", serveCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("port", serveCmd.PersistentFlags().Lookup("port"))
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Runs the search API server",
	Long:  `Runs the search API server`,
	Args:  cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("codeprofile") {
			defer profile.Start(profile.NoShutdownHook).Stop()
		}
		config.InitializeConfiguration()
		actions.AutoUpdateCommand = "" //config.GetAutoUpdateCommand()
		//Background Cron Jobs
		jobs.Start()
		app.DoYourThing()
	},
}
