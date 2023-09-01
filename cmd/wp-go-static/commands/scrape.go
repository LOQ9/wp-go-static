package commands

import (
	"crypto/tls"
	"fmt"
	"log"
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

type ScrapeConfig struct {
	Dir        string `mapstructure:"dir"`
	URL        string `mapstructure:"url"`
	Cache      string `mapstructure:"cache"`
	ReplaceURL string `mapstructure:"replace-url"`
	Replace    bool   `mapstructure:"replace"`
	Parallel   bool   `mapstructure:"parallel"`
	Images     bool   `mapstructure:"images"`
	CheckHead  bool   `mapstructure:"check-head"`
}

type Scrape struct {
	urlCache *URLCache
	c        *colly.Collector
	domain   string
	hostname string
	config   ScrapeConfig
}

func NewScrape() *Scrape {
	return &Scrape{
		urlCache: &URLCache{urls: make(map[string]bool)},
		c:        colly.NewCollector(),
	}
}

// ScrapeCmd ...
var ScrapeCmd = &cobra.Command{
	Use:   "scrape",
	Short: "Scrape from the Wordpress website",
	RunE:  scrapeCmdF,
}

func init() {
	// Define command-line flags
	ScrapeCmd.PersistentFlags().String("dir", "dump", "directory to save downloaded files")
	ScrapeCmd.PersistentFlags().String("url", "", "URL to scrape")
	ScrapeCmd.PersistentFlags().String("cache", "", "Cache directory")
	ScrapeCmd.PersistentFlags().String("replace-url", "", "Replace with a specific url")
	ScrapeCmd.PersistentFlags().Bool("replace", true, "Replace url")
	ScrapeCmd.PersistentFlags().Bool("parallel", false, "Fetch in parallel")
	ScrapeCmd.PersistentFlags().Bool("images", true, "Download images")
	ScrapeCmd.PersistentFlags().Bool("check-head", true, "Checks head")
	ScrapeCmd.MarkFlagRequired("url")

	// Bind command-line flags to Viper
	err := viper.BindPFlags(ScrapeCmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}

	RootCmd.AddCommand(ScrapeCmd)
}

func scrapeCmdF(command *cobra.Command, args []string) error {
	scrape := NewScrape()
	viper.Unmarshal(&scrape.config)

	scrape.domain = scrape.config.URL

	if scrape.config.CheckHead {
		scrape.c.CheckHead = true
	}

	if scrape.config.Cache != "" {
		log.Println("Using cache directory", scrape.config.Cache)
		scrape.c.CacheDir = scrape.config.Cache
	}

	scrape.c.Async = scrape.config.Parallel

	// Use a custom TLS config to verify server certificates
	scrape.c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{},
	})

	parsedURL, err := url.Parse(scrape.config.URL)
	if err != nil {
		return err
	}
	scrape.hostname = parsedURL.Hostname()

	// Visit only pages that are part of the website
	scrape.c.AllowedDomains = []string{scrape.hostname}

	// On every a element which has href attribute call callback
	scrape.c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		scrape.visitURL(link)
	})

	// On every link element call callback
	scrape.c.OnHTML("link[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		scrape.visitURL(link)
	})

	// On every script element call callback
	scrape.c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		scrape.visitURL(link)
	})

	// On every img element call callback
	scrape.c.OnHTML("img", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		srcSet := e.Attr("srcset")
		scrape.visitURL(src)

		if srcSet != "" {
			srcSetList := strings.Split(srcSet, ",")
			for _, srcSetURL := range srcSetList {
				srcSetURL = strings.TrimSpace(srcSetURL)
				innerSrcSet := strings.Split(srcSetURL, " ")[0]
				if innerSrcSet == "" {
					continue
				}

				scrape.visitURL(innerSrcSet)
			}
		}
	})

	// Before making a request print "Visiting ..."
	scrape.c.OnRequest(func(r *colly.Request) {
		switch r.Method {
		case http.MethodGet:
			log.Printf("Visiting: %s\n", r.URL.String())
		case http.MethodHead:
			log.Printf("Checking: %s\n", r.URL.String())
		default:
			log.Printf("Skipping [%s]: %s\n", r.Method, r.URL.String())
		}
	})

	// On response
	scrape.c.OnResponse(func(r *colly.Response) {
		rCopy := *r
		dir, fileName := file.HandleFile(r, scrape.config.Dir)
		rCopy.Body = scrape.parseBody(r.Body)

		err := file.SaveFile(&rCopy, dir, fileName)
		if err != nil {
			log.Println(err)
			return
		}
	})

	urlsToVisit := []string{
		"robots.txt",
		// "sitemap.xml",
		"favicon.ico",
	}

	for _, domain := range urlsToVisit {
		err = scrape.c.Visit(scrape.domain + "/" + domain)
		if err != nil {
			log.Println(err)
		}
	}

	// Start scraping
	err = scrape.c.Visit(scrape.domain)

	if err != nil {
		return err
	}

	scrape.c.Wait()

	return nil
}

func (s *Scrape) visitURL(link string) {
	link = s.getAbsoluteURL(link)

	if link == "" {
		return
	}

	u, err := url.Parse(link)
	if err != nil {
		log.Printf("Error parsing URL %s: %s", link, err)
		return
	}

	if u.Scheme == "" || u.Host == "" {
		log.Printf("Invalid URL %s", link)
		return
	}

	u.Fragment = ""

	link = u.String()

	// Download page if it hasn't been visited before
	if !s.urlCache.Get(link) {
		s.urlCache.Add(link)
		err := s.c.Visit(link)
		if err != nil {
			log.Println(err)
		}
	}
}

func (s *Scrape) parseBody(body []byte) []byte {
	cssUrls := regexp.MustCompile(`url\((https?://[^\s]+)\)`).FindAllStringSubmatch(string(body), -1)

	// Download each referenced file if it hasn't been visited before
	for _, cssUrl := range cssUrls {
		link := strings.Trim(cssUrl[1], "'\"")
		if link == "" {
			continue
		}
		s.visitURL(link)
	}

	if s.config.Replace {
		optionList := []string{
			fmt.Sprintf(`http://%s`, s.hostname),
			fmt.Sprintf(`http:\/\/%s`, s.hostname),
			fmt.Sprintf(`https://%s`, s.hostname),
			fmt.Sprintf(`https:\/\/%s`, s.hostname),
		}

		for _, option := range optionList {
			// Replace all occurrences of the base URL with a relative URL
			replaceBody := strings.ReplaceAll(string(body), option, s.config.ReplaceURL)
			body = []byte(replaceBody)
		}
	}

	return body
}

func (s *Scrape) getAbsoluteURL(inputURL string) string {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return ""
	}

	if parsedURL.IsAbs() {
		return inputURL
	}

	baseURL, err := url.Parse(s.domain)
	if err != nil {
		return ""
	}

	return baseURL.ResolveReference(parsedURL).String()
}