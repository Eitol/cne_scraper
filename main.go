package main

import (
	"github.com/Eitol/cne_scraper/scraper"
)

func main() {
	s := scraper.BuildScraper()
	s.Scrap()
}
