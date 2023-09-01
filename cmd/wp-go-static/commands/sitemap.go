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

	for i := range smap.URL {
		// Replace the URL with the url from the replace-url argument
		// Only with the URL part, persist the URL path and query
		if config.Sitemap.ReplaceURL != "" {
			currentURL, _ := url.Parse(config.Sitemap.URL)

			optionList := []string{
				fmt.Sprintf(`http://%s`, currentURL.Host),
				fmt.Sprintf(`http:\/\/%s`, currentURL.Host),
				fmt.Sprintf(`https://%s`, currentURL.Host),
				fmt.Sprintf(`https:\/\/%s`, currentURL.Host),
			}

			for _, option := range optionList {
				if i >= len(smap.URL) {
					fmt.Println("Index out of range for smap.URL")
					break
				}
				smap.URL[i].Loc = strings.ReplaceAll(string(smap.URL[i].Loc), option, config.Sitemap.ReplaceURL)

				// for j := range smap.Image {
				// 	if i >= len(smap.URL) {
				// 		fmt.Println("Index out of range for smap.URL")
				// 		break
				// 	}
				// 	if j >= len(smap.URL[i].Image) {
				// 		fmt.Println("Index out of range for smap.URL[i].Image")
				// 		break
				// 	}
				// 	smap.URL[i].Image[j].Loc = strings.ReplaceAll(string(smap.URL[i].Image[j].Loc), option, sitemapConfig.ReplaceURL)
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
