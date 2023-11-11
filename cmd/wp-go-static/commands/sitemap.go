package commands

import (
	"fmt"
	"net/url"
	"strings"

	"wp-go-static/internal/config"
	goSitemap "wp-go-static/pkg/sitemap"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// SitemapCmd ...
var SitemapCmd = &cobra.Command{
	Use:   "sitemap",
	Short: "Create sitemap from the Wordpress website",
	RunE:  sitemapCmdF,
}

const (
	bindFlagSitemapPrefix = "sitemap"
)

func init() {
	// Define command-line flags
	SitemapCmd.PersistentFlags().String("dir", "dump", "directory to save downloaded files")
	SitemapCmd.PersistentFlags().String("url", "", "URL to scrape")
	SitemapCmd.PersistentFlags().String("replace-url", "", "Replace with a specific url")
	SitemapCmd.PersistentFlags().String("file", "sitemap.xml", "Output sitemap file name")

	// Bind command-line flags to Viper
	SitemapCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		bindFlag := fmt.Sprintf("%s.%s", bindFlagSitemapPrefix, flag.Name)
		viper.BindPFlag(bindFlag, SitemapCmd.PersistentFlags().Lookup(flag.Name))
	})

	RootCmd.AddCommand(SitemapCmd)
}

func sitemapCmdF(command *cobra.Command, args []string) error {
	config := config.Config{}
	viper.Unmarshal(&config)

	smap, err := goSitemap.Get(config.Sitemap.URL, nil)
	if err != nil {
		fmt.Println(err)
	}

	if config.Sitemap.ReplaceURL != "" {
		currentURL, _ := url.Parse(config.Sitemap.URL)
		host := currentURL.Host
		optionList := []string{
			"http://" + host,
			"http:\\/\\/" + host,
			"https://" + host,
			"https:\\/\\/" + host,
		}

		for i := range smap.URL {
			for _, option := range optionList {
				smap.URL[i].Loc = strings.ReplaceAll(smap.URL[i].Loc, option, config.Sitemap.ReplaceURL)

				// for j := range smap.URL[i].Image {
				// 	smap.URL[i].Image[j].Loc = strings.ReplaceAll(smap.URL[i].Image[j].Loc, option, config.Sitemap.ReplaceURL)
				// }
			}
		}
	}

	// Print the Sitemap
	printSmap, err := smap.Print()
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", printSmap)

	// Write the Sitemap to a file
	if config.Sitemap.File != "" {
		fmt.Printf("Writing sitemap to %s/%s\n", config.Sitemap.Dir, config.Sitemap.File)
		return smap.Save(config.Sitemap.Dir, config.Sitemap.File)
	}

	return nil
}
