package config

type Config struct {
	Scrape  ScrapeConfig  `mapstructure:"scrape"`
	Sitemap SitemapConfig `mapstructure:"sitemap"`
	Robots  RobotsConfig  `mapstructure:"robots"`
}

type SitemapConfig struct {
	Dir        string            `mapstructure:"dir"`
	URL        string            `mapstructure:"url"`
	ReplaceURL string            `mapstructure:"replace-url"`
	File       string            `mapstructure:"file"`
	Headers    map[string]string `mapstructure:"headers"`
}

type ScrapeConfig struct {
	Dir        string            `mapstructure:"dir"`
	URL        string            `mapstructure:"url"`
	Cache      string            `mapstructure:"cache"`
	ReplaceURL string            `mapstructure:"replace-url"`
	Replace    bool              `mapstructure:"replace"`
	Parallel   bool              `mapstructure:"parallel"`
	Images     bool              `mapstructure:"images"`
	CheckHead  bool              `mapstructure:"check-head"`
	ExtraPages []string          `mapstructure:"extra-pages"`
	Headers    map[string]string `mapstructure:"headers"`
}

type RobotsConfig struct {
	Dir        string            `mapstructure:"dir"`
	URL        string            `mapstructure:"url"`
	ReplaceURL string            `mapstructure:"replace-url"`
	File       string            `mapstructure:"file"`
	Headers    map[string]string `mapstructure:"headers"`
}
