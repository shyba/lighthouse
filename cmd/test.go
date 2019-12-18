package cmd

import (
	"os"
	"time"

	"github.com/lbryio/lighthouse/app/test"

	"github.com/lbryio/lighthouse/app"
	"github.com/lbryio/lighthouse/app/actions"
	"github.com/lbryio/lighthouse/app/config"
	"github.com/lbryio/lighthouse/app/jobs"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	testCmd.PersistentFlags().String("host", "0.0.0.0", "host to listen on")
	testCmd.PersistentFlags().IntP("port", "p", 50005, "port binding used for the api server")
	testCmd.PersistentFlags().StringArrayP("channels", "c", []string{}, "the channels the test should sync")
	//Bind to Viper
	viper.BindPFlags(testCmd.PersistentFlags())
	rootCmd.AddCommand(testCmd)
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Runs the search API server in test mode",
	Long:  `Runs the search API server in test mode`,
	Args:  cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("codeprofile") {
			defer profile.Start(profile.NoShutdownHook).Stop()
		}
		config.InitializeConfiguration()
		actions.AutoUpdateCommand = "" //config.GetAutoUpdateCommand()
		//Background Cron Jobs
		channels, err := cmd.Flags().GetStringArray("channels")
		if err != nil {
			panic(err)
		}
		for _, c := range channels {
			jobs.SyncClaims(&c)
		}

		go app.DoYourThing()
		time.Sleep(1 * time.Second)
		test.RunTests()
		os.Exit(0)
	},
}
