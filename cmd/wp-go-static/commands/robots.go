package commands

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"wp-go-static/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// RobotsCmd ...
var RobotsCmd = &cobra.Command{
	Use:   "robots",
	Short: "Create robots from the Wordpress website",
	RunE:  robotsCmdF,
}

const (
	bindFlagRobotsPrefix = "robots"
)

func init() {
	// Define command-line flags
	RobotsCmd.PersistentFlags().String("dir", "dump", "directory to save downloaded files")
	RobotsCmd.PersistentFlags().String("url", "", "URL to scrape")
	RobotsCmd.PersistentFlags().String("replace-url", "", "Replace with a specific url")
	RobotsCmd.PersistentFlags().String("file", "robots.txt", "Output robots file name")

	// Bind command-line flags to Viper
	RobotsCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		bindFlag := fmt.Sprintf("%s.%s", bindFlagRobotsPrefix, flag.Name)
		viper.BindPFlag(bindFlag, RobotsCmd.PersistentFlags().Lookup(flag.Name))
	})

	RootCmd.AddCommand(RobotsCmd)
}

func robotsCmdF(command *cobra.Command, args []string) error {
	config := config.Config{}
	viper.Unmarshal(&config)

	// Fetch the robots.txt from the provided URL
	resp, err := http.Get(config.Robots.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body into a string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	modifiedBody := string(body)

	if config.Robots.ReplaceURL != "" {
		// Perform the search and replace operation
		modifiedBody = strings.ReplaceAll(string(body), config.Robots.URL, config.Robots.ReplaceURL)
	}

	if config.Robots.ReplaceURL != "" {
		currentURL, _ := url.Parse(config.Robots.URL)
		host := currentURL.Host
		optionList := []string{
			"http://" + host,
			"http:\\/\\/" + host,
			"https://" + host,
			"https:\\/\\/" + host,
		}

		for _, option := range optionList {
			// Find the matching string
			if !strings.Contains(modifiedBody, option) {
				continue
			}

			// Replace the string
			modifiedBody = strings.ReplaceAll(modifiedBody, option, config.Robots.ReplaceURL)
		}
	}

	// Print the output
	fmt.Println(modifiedBody)

	// Create a new file
	out, err := os.Create(filepath.Join(config.Robots.Dir, config.Robots.File))
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the modified string to the new file
	if _, err := out.WriteString(modifiedBody); err != nil {
		return err
	}

	return nil
}
