package commands

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"wp-go-static/pkg/file"

	"github.com/gocolly/colly"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"wp-go-static/internal/cache"
	"wp-go-static/internal/config"
)

type Scrape struct {
	urlCache *cache.URLCache
	c        *colly.Collector
	domain   string
	hostname string
	config   config.Config
}

func NewScrape() *Scrape {
	return &Scrape{
		urlCache: &cache.URLCache{URLs: make(map[string]bool)},
		c:        colly.NewCollector(),
	}
}

// ScrapeCmd ...
var ScrapeCmd = &cobra.Command{
	Use:   "scrape",
	Short: "Scrape from the Wordpress website",
	RunE:  scrapeCmdF,
}

const (
	bindFlagScrapePrefix = "scrape"
)

func init() {
	// Define command-line flags
	ScrapeCmd.PersistentFlags().String("dir", "dump", "directory to save downloaded files")
	ScrapeCmd.PersistentFlags().String("url", "", "URL to scrape")
	ScrapeCmd.PersistentFlags().String("cache", "", "Cache directory")
	ScrapeCmd.PersistentFlags().String("replace-url", "", "Replace with a specific url")
	ScrapeCmd.PersistentFlags().StringSlice("extra-pages", []string{}, "Extra pages to scrape")
	ScrapeCmd.PersistentFlags().Bool("replace", true, "Replace url")
	ScrapeCmd.PersistentFlags().Bool("parallel", false, "Fetch in parallel")
	ScrapeCmd.PersistentFlags().Bool("images", true, "Download images")
	ScrapeCmd.PersistentFlags().Bool("check-head", true, "Checks head")
	// ScrapeCmd.MarkPersistentFlagRequired("url")

	ScrapeCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		bindFlag := fmt.Sprintf("%s.%s", bindFlagScrapePrefix, flag.Name)
		viper.BindPFlag(bindFlag, ScrapeCmd.PersistentFlags().Lookup(flag.Name))
	})

	RootCmd.AddCommand(ScrapeCmd)
}

func scrapeCmdF(command *cobra.Command, args []string) error {
	scrape := NewScrape()
	viper.Unmarshal(&scrape.config)

	scrape.domain = scrape.config.Scrape.URL

	if scrape.config.Scrape.CheckHead {
		scrape.c.CheckHead = true
	}

	if scrape.config.Scrape.Cache != "" {
		log.Println("Using cache directory", scrape.config.Scrape.Cache)
		scrape.c.CacheDir = scrape.config.Scrape.Cache
	}

	scrape.c.Async = scrape.config.Scrape.Parallel

	// Use a custom TLS config to verify server certificates
	scrape.c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{},
	})

	parsedURL, err := url.Parse(scrape.config.Scrape.URL)
	if err != nil {
		return err
	}
	scrape.hostname = parsedURL.Hostname()

	// Visit only pages that are part of the website
	scrape.c.AllowedDomains = []string{scrape.hostname}

	for _, extraPage := range scrape.config.Scrape.ExtraPages {
		log.Println("Visiting Extra Page:", extraPage)
		scrape.visitURL(extraPage)
	}

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
		dir, fileName := file.HandleFile(r, scrape.config.Scrape.Dir)
		rCopy.Body = scrape.parseBody(r.Body)

		err := file.SaveFile(&rCopy, dir, fileName)
		if err != nil {
			log.Println(err)
			return
		}
	})

	urlsToVisit := []string{
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
		log.Printf("Invalid URL: %s", link)
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

	if s.config.Scrape.Replace {
		optionList := []string{
			fmt.Sprintf(`http://%s`, s.hostname),
			fmt.Sprintf(`http:\/\/%s`, s.hostname),
			fmt.Sprintf(`https://%s`, s.hostname),
			fmt.Sprintf(`https:\/\/%s`, s.hostname),
		}

		for _, option := range optionList {
			// Replace all occurrences of the base URL with a relative URL
			replaceBody := strings.ReplaceAll(string(body), option, s.config.Scrape.ReplaceURL)
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
