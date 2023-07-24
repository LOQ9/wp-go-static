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
	defer c.mu.Unlock()
	_, ok := c.urls[url]
	return ok
}

type Scrape struct {
	urlCache *URLCache
	c        *colly.Collector
	domain   string
}

func NewScrape() *Scrape {
	return &Scrape{
		urlCache: &URLCache{urls: make(map[string]bool)},
		c:        colly.NewCollector(),
	}
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
	RootCmd.PersistentFlags().Bool("parallel", false, "Fetch in parallel")
	RootCmd.PersistentFlags().Bool("images", true, "Download images")

	// Bind command-line flags to Viper
	viper.BindPFlag("dir", RootCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("url", RootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("cache", RootCmd.PersistentFlags().Lookup("cache"))
	viper.BindPFlag("images", RootCmd.PersistentFlags().Lookup("images"))

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
	parallel := viper.GetBool("parallel")

	scrape := NewScrape()

	scrape.domain = commandURL

	if cacheDir != "" {
		scrape.c.CacheDir = cacheDir
	}

	scrape.c.Async = parallel

	// Ignore SSL errors
	scrape.c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

	parsedURL, err := url.Parse(commandURL)
	if err != nil {
		return err
	}
	domain := parsedURL.Hostname()

	// Visit only pages that are part of the website
	scrape.c.AllowedDomains = []string{domain}

	// On every a element which has href attribute call callback
	scrape.c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		scrape.visitURL(e.Request.AbsoluteURL(link))
	})

	// On every link element call callback
	scrape.c.OnHTML("link[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		scrape.visitURL(e.Request.AbsoluteURL(link))
	})

	// On every script element call callback
	scrape.c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		scrape.visitURL(e.Request.AbsoluteURL(link))
	})

	// On every img element call callback

	scrape.c.OnHTML("img", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		scrape.visitURL(e.Request.AbsoluteURL(link))
	})

	// Before making a request print "Visiting ..."
	scrape.c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// On response
	scrape.c.OnResponse(func(r *colly.Response) {
		dir, fileName := file.HandleFile(r, commandDir)
		r.Body = scrape.parseBody(r.Body)

		err := file.SaveFile(r, dir, fileName)
		if err != nil {
			fmt.Println(err)
			return
		}
	})

	// Start scraping
	err = scrape.c.Visit(commandURL)

	if err != nil {
		return err
	}

	scrape.c.Wait()

	return nil
}

func (s *Scrape) visitURL(link string) {
	// Download image found on page if it hasn't been visited before
	if !s.urlCache.Get(link) {
		s.urlCache.Add(link)
		s.c.Visit(link)
	}
}

func (s *Scrape) parseBody(body []byte) []byte {
	// Find all URLs in the CSS file
	cssUrls := regexp.MustCompile(`url\((https?://[^\s]+)\)`).FindAllStringSubmatch(string(body), -1)

	// Download each referenced file if it hasn't been visited before
	for _, cssUrl := range cssUrls {
		link := strings.Trim(cssUrl[1], "'\"")
		if link == "" {
			continue
		}
		s.visitURL(link)
	}

	optionList := []string{
		fmt.Sprintf(`http://%s`, s.domain),
		fmt.Sprintf(`http:\/\/%s`, s.domain),
		fmt.Sprintf(`https://%s`, s.domain),
		fmt.Sprintf(`https:\/\/%s`, s.domain),
		s.domain,
	}

	for _, v := range optionList {
		// Replace all occurrences of the base URL with a relative URL
		replaceBody := strings.ReplaceAll(string(body), v, "")
		body = []byte(replaceBody)
	}

	return body
}
