package tracker

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

type CollyTracker struct{}

type Bar struct {
	Name      string `json:"bar"`
	Address   string `json:"adres"`
	Url       string `json:"url"`
	Beers     []Beer `json:"piwa"`
	ScrapeErr error  `json:"errors"`
}

type Beer struct {
	Name   string
	Prices string
}

func NewCollyTracker() *CollyTracker {
	return &CollyTracker{}
}

func (ct CollyTracker) newCollector() *colly.Collector {
	return colly.NewCollector(colly.AllowedDomains())
}

func (ct CollyTracker) GetBarsWithWellPricedBeers(priceLimit int) ([]Bar, error) {
	var barsWithWellPricedBeers []Bar
	bars := make(chan Bar)
	barsUrls, err := ct.FetchBarsUrls()
	if err != nil {
		log.Print("ERROR during fetching bars urls: ", err)
		return []Bar{}, err
	}

	wg := &sync.WaitGroup{}
	for _, url := range barsUrls {
		wg.Add(1)
		go ct.FetchBarInfo(wg, url, bars)
	}

	go func() {
		wg.Wait()
		close(bars)
	}()

	for bar := range bars {
		beers, err := bar.SearchForWellPricedBeers(priceLimit, bar)
		if err != nil {
			log.Print("ERROR during searching for well priced beers: ", err)
			return []Bar{}, err
		}
		if len(beers) > 0 {
			bar.Beers = beers
			barsWithWellPricedBeers = append(barsWithWellPricedBeers, bar)
		}
	}

	return barsWithWellPricedBeers, nil
}

func (ct CollyTracker) FetchBarsUrls() ([]string, error) {
	var urls []string
	var scrapeErr error

	c := ct.newCollector()

	c.OnError(func(_ *colly.Response, err error) {
		scrapeErr = err
		log.Print("Something went wrong: ", err)
	})

	c.OnHTML("div.panel.panel-default.text-center", func(e *colly.HTMLElement) {
		var url string

		e.DOM.Find("a").Each(func(_ int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists {
				url = e.Request.AbsoluteURL(href)
			}
		})
		urls = append(urls, url)
	})

	if err := c.Visit("https://ontap.pl/warszawa/multitaps"); err != nil {
		return nil, err
	}

	if scrapeErr != nil {
		return nil, scrapeErr
	}
	return urls, nil
}

func (ct CollyTracker) FetchBarInfo(wg *sync.WaitGroup, barUrl string, bar chan Bar) {
	defer wg.Done()
	var name, address string
	var beers []Beer
	var scrapeErr error

	c := ct.newCollector()

	c.OnError(func(_ *colly.Response, err error) {
		scrapeErr = err
		log.Print("Something went wrong: ", err)
	})

	c.OnHTML("ol.breadcrumb li.active", func(e *colly.HTMLElement) {
		name = strings.TrimSpace(e.Text)
	})

	c.OnHTML("div.panel.panel-default", func(e *colly.HTMLElement) {
		var beerName, prices string
		e.DOM.Find("h4.cml_shadow").Each(func(_ int, s *goquery.Selection) {
			beerName = strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), "\t", "")
		})
		e.DOM.Find("div.col-xs-7").Each(func(_ int, s *goquery.Selection) {
			prices = strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), "\t", "")
		})
		beers = append(beers, Beer{
			Name:   beerName,
			Prices: prices,
		})
	})

	c.OnHTML("div.text-left", func(e *colly.HTMLElement) {
		e.DOM.Contents().Each(func(_ int, s *goquery.Selection) {
			if goquery.NodeName(s) == "i" {
				iconClass, _ := s.Attr("class")
				if strings.Contains(iconClass, "fa-map-marker") {
					address = strings.TrimSpace(s.Parent().Contents().Get(2).Data)
				}
			}
		})
	})

	c.OnScraped(func(r *colly.Response) {
		bar <- Bar{name, address, barUrl, beers, scrapeErr}
	})

	if err := c.Visit(barUrl); err != nil {
		scrapeErr = err
	}
}

func (b Bar) SearchForWellPricedBeers(priceLimit int, bar Bar) ([]Beer, error) {
	var wellPricedBeers []Beer
	var searchErrors []error
	for _, beer := range bar.Beers {
		for _, priceStr := range strings.Split(beer.Prices, " · ") {
			if strings.Contains(priceStr, "0.5l:") {
				price, err := strconv.Atoi(strings.Replace(strings.Split(priceStr, ": ")[1], "zł", "", 1))
				if err != nil {
					searchErrors = append(searchErrors, err)
				}
				if price < priceLimit {
					wellPricedBeers = append(wellPricedBeers, beer)
				}
			}
		}
	}
	if len(searchErrors) > 0 {
		return nil, fmt.Errorf("errors occured during search for well priced beers: %v", searchErrors)
	}
	return wellPricedBeers, nil
}
