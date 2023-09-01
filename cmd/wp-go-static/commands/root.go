package commands

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Run ...
func Run(args []string) error {
	RootCmd.SetArgs(args)
	return RootCmd.Execute()
}

// RootCmd ..
var RootCmd = &cobra.Command{
	Use:   "wp-go-static",
	Short: "Wordpress Go Static",
	Long:  `Wordpress Go Static is a tool to download a Wordpress website and make it static`,
}

func init() {
	err := viper.BindPFlags(RootCmd.PersistentFlags())
	if err != nil {
		panic(err)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.SetEnvPrefix("WGS")
	viper.AutomaticEnv()
}
