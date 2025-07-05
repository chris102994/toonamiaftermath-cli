package run

type Run struct {
	XMLTVOutput string `mapstructure:"xmltv_output,omitempty"`
	M3UOutput   string `mapstructure:"m3u_output,omitempty"`
	CacheFile   string `mapstructure:"cache_file,omitempty"`
	ScrapeCount int    `mapstructure:"scrape_count,omitempty"`
}
