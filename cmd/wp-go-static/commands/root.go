package commands

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"wp-go-static/pkg/file"

	"github.com/gocolly/colly"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// URLCache is a struct to hold the visited URLs
type URLCache struct {
	mu   sync.Mutex
	urls map[string]bool
}

// Add adds a URL to the cache
func (c *URLCache) Add(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.urls[url] = true
}

// Get checks if a URL is in the cache
func (c *URLCache) Get(url string) bool {
	c.mu.Lock()
	defer c.mu.Unl
	ock()
	_, ok := c.urls[url]
	return ok
}

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
	RunE:  rootCmdF,
}

func init() {
	// Define command-line flags
	RootCmd.PersistentFlags().String("dir", "dump", "directory to save downloaded files")
	RootCmd.PersistentFlags().String("url", "", "URL to scrape")
	RootCmd.PersistentFlags().String("cache", "", "Cache directory")

	// Bind command-line flags to Viper
	viper.BindPFlag("dir", RootCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("url", RootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("cache", RootCmd.PersistentFlags().Lookup("cache"))

	viper.AutomaticEnv()
	viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("WGS")

	// Execute root command
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func rootCmdF(command *cobra.Command, args []string) error {
	commandDir := viper.GetString("dir")
	commandURL := viper.GetString("url")
	cacheDir := viper.GetString("cache")

	c := colly.NewCollector()

	if cacheDir != "" {
		c.CacheDir = cacheDir
	}

	// Ignore SSL errors
	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

	parsedURL, err := url.Parse(commandURL)
	if err != nil {
		return err
	}
	domain := parsedURL.Hostname()

	// Visit only pages that are part of the website
	c.AllowedDomains = []string{domain}

	// Create URL cache
	urlCache := &URLCache{urls: make(map[string]bool)}

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Visit link found on page if it hasn't been visited before
		if !urlCache.Get(link) {
			urlCache.Add(link)
			c.Visit(e.Request.AbsoluteURL(link))
		}
	})

	// On every link element call callback
	c.OnHTML("link[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Download file found on page if it has a supported extension and hasn't been visited before
		if !urlCache.Get(link) {
			urlCache.Add(link)
			c.Visit(e.Request.AbsoluteURL(link))
		}
	})

	// On every script element call callback
	c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		// Download file found on page if it has a supported extension and hasn't been visited before
		if !urlCache.Get(link) {
			urlCache.Add(link)
			c.Visit(e.Request.AbsoluteURL(link))
		}
	})

	// On every img element call callback
	c.OnHTML("img", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		// Download image found on page if it hasn't been visited before
		if !urlCache.Get(link) {
			urlCache.Add(link)
			c.Visit(e.Request.AbsoluteURL(link))
		}
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// On response
	c.OnResponse(func(r *colly.Response) {
		dir, fileName := file.HandleFile(r, commandDir)

		// Find all URLs in the CSS file
		cssUrls := regexp.MustCompile(`url\((https?://[^\s]+)\)`).FindAllStringSubmatch(string(r.Body), -1)

		// Download each referenced file if it hasn't been visited before
		for _, cssUrl := range cssUrls {
			url := strings.Trim(cssUrl[1], "'\"")
			if url == "" {
				continue
			}
			if !urlCache.Get(url) {
				urlCache.Add(url)
				fmt.Printf("Visiting from CSS: '%s'\n", url)
				c.Visit(url)
			}
		}

		optionList := []string{
			fmt.Sprintf(`http://%s`, domain),
			fmt.Sprintf(`http:\/\/%s`, domain),
			fmt.Sprintf(`https://%s`, domain),
			fmt.Sprintf(`https:\/\/%s`, domain),
			domain,
		}

		for _, v := range optionList {
			// Replace all occurrences of the base URL with a relative URL
			replaceBody := strings.ReplaceAll(string(r.Body), v, "")
			r.Body = []byte(replaceBody)
		}

		err := file.SaveFile(r, dir, fileName)
		if err != nil {
			fmt.Println(err)
			return
		}
	})

	// Start scraping
	return c.Visit(commandURL)
}
