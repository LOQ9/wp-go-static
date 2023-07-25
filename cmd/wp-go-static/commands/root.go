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
	RootCmd.MarkFlagRequired("url")

	// Bind command-line flags to Viper
	err := viper.BindPFlags(RootCmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}

	viper.AutomaticEnv()
	viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("WGS")
}

func rootCmdF(command *cobra.Command, args []string) error {
	commandDir := viper.GetString("dir")
	commandURL := viper.GetString("url")
	cacheDir := viper.GetString("cache")
	parallel := viper.GetBool("parallel")

	scrape := NewScrape()

	scrape.domain = commandURL

	if cacheDir != "" {
		log.Println("Using cache directory", cacheDir)
		scrape.c.CacheDir = cacheDir
	}

	scrape.c.Async = parallel

	// Use a custom TLS config to verify server certificates
	scrape.c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{},
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
		log.Println("Visiting", r.URL.String())
	})

	// On response
	scrape.c.OnResponse(func(r *colly.Response) {
		dir, fileName := file.HandleFile(r, commandDir)
		r.Body = scrape.parseBody(r.Body)

		err := file.SaveFile(r, dir, fileName)
		if err != nil {
			log.Println(err)
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
