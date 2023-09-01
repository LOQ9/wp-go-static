package commands

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	goSitemap "wp-go-static/pkg/sitemap"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type SitemapConfig struct {
	Dir         string `mapstructure:"dir"`
	URL         string `mapstructure:"url"`
	ReplaceURL  string `mapstructure:"replace-url"`
	SitemapFile string `mapstructure:"sitemap-file"`
}

// SitemapCmd ...
var SitemapCmd = &cobra.Command{
	Use:   "sitemap",
	Short: "Create sitemap from the Wordpress website",
	RunE:  sitemapCmdF,
}

func init() {
	// Define command-line flags
	SitemapCmd.PersistentFlags().String("dir", "dump", "directory to save downloaded files")
	SitemapCmd.PersistentFlags().String("url", "", "URL to scrape")
	SitemapCmd.PersistentFlags().String("replace-url", "", "Replace with a specific url")
	SitemapCmd.PersistentFlags().String("sitemap-file", "sitemap.xml", "Output sitemap file name")
	SitemapCmd.MarkFlagRequired("url")

	// Bind command-line flags to Viper
	err := viper.BindPFlags(SitemapCmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}

	RootCmd.AddCommand(SitemapCmd)
}

func sitemapCmdF(command *cobra.Command, args []string) error {
	sitemapConfig := SitemapConfig{}
	viper.Unmarshal(&sitemapConfig)

	smap, err := goSitemap.Get(sitemapConfig.URL, nil)
	if err != nil {
		fmt.Println(err)
	}

	for i := range smap.URL {
		// Replace the URL with the url from the replace-url argument
		// Only with the URL part, persist the URL path and query
		if sitemapConfig.ReplaceURL != "" {
			currentURL, _ := url.Parse(sitemapConfig.URL)

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
				smap.URL[i].Loc = strings.ReplaceAll(string(smap.URL[i].Loc), option, sitemapConfig.ReplaceURL)

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
	if sitemapConfig.SitemapFile != "" {
		fmt.Printf("Writing sitemap to %s/%s\n", sitemapConfig.Dir, sitemapConfig.SitemapFile)
		return smap.Save(sitemapConfig.Dir, sitemapConfig.SitemapFile)
	}

	return nil
}
