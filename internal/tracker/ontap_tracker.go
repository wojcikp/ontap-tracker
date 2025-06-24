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

const PriceLimit = 18

type Tracker interface {
	FetchBarsInWarsaw() []Bar
}

type CollyTracker struct{}

type Bar struct {
	Name      string
	Url       string
	Beers     *[]Beer
	ScrapeErr error
}

type Beer struct {
	Name   string
	Prices string
}

type BarWithWellPricedBeers struct {
	Bar     string   `json:bar`
	Address string   `json:adres`
	Beers   []Beer   `json:piwa`
	Errors  []string `json:errors`
}

func NewCollyTracker() *CollyTracker {
	return &CollyTracker{}
}

func (ct CollyTracker) newCollector() *colly.Collector {
	return colly.NewCollector(colly.AllowedDomains())
}

func (ct CollyTracker) FetchBarsInWarsaw() ([]Bar, error) {
	var bars []Bar
	var scrapeErr error

	c := ct.newCollector()

	c.OnError(func(_ *colly.Response, err error) {
		scrapeErr = err
		log.Print("Something went wrong: ", err)
	})

	c.OnHTML("div.panel.panel-default.text-center", func(e *colly.HTMLElement) {
		name := strings.Split(strings.Replace(strings.ReplaceAll(strings.TrimPrefix(e.Text, "\n"), "\t", ""), "\n\n", "", -1), "\n")[0]
		var url string

		e.DOM.Find("a").Each(func(_ int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists {
				url = e.Request.AbsoluteURL(href)
			}
		})
		bars = append(bars, Bar{
			Name:  name,
			Url:   url,
			Beers: &[]Beer{},
		})
	})

	if err := c.Visit("https://ontap.pl/warszawa/multitaps"); err != nil {
		return nil, err
	}

	if scrapeErr != nil {
		return nil, scrapeErr
	}
	return bars, nil
}

func (ct CollyTracker) FetchBeersInfo(wg *sync.WaitGroup, bar *Bar) {
	defer wg.Done()

	var beers []Beer

	c := ct.newCollector()

	c.OnError(func(_ *colly.Response, err error) {
		bar.ScrapeErr = err
		log.Print("Something went wrong: ", err)
	})

	c.OnHTML("div.panel.panel-default", func(e *colly.HTMLElement) {
		var name, prices string
		e.DOM.Find("h4.cml_shadow").Each(func(_ int, s *goquery.Selection) {
			name = strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), "\t", "")
		})
		e.DOM.Find("div.col-xs-7").Each(func(_ int, s *goquery.Selection) {
			prices = strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), "\t", "")
		})
		beers = append(beers, Beer{
			Name:   name,
			Prices: prices,
		})
	})

	c.OnScraped(func(r *colly.Response) {
		*bar.Beers = beers
	})

	if err := c.Visit(bar.Url); err != nil {
		bar.ScrapeErr = err
	}
}

func (ct CollyTracker) GetBeersInfo() ([]BarWithWellPricedBeers, error) {
	var barsWithGoodPrices []BarWithWellPricedBeers

	bars, err := ct.FetchBarsInWarsaw()
	if err != nil {
		return nil, err
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(bars))
	for _, bar := range bars {
		go ct.FetchBeersInfo(wg, &bar)
	}
	wg.Wait()

	for _, bar := range bars {
		var errors []string
		beerInfo := BarWithWellPricedBeers{Bar: bar.Name, Address: "ul. testowa 999"}
		if bar.ScrapeErr != nil {
			log.Printf("Scrape error in bar: %s \nERROR: %v", bar.Name, bar.ScrapeErr)
			errors = append(errors, bar.ScrapeErr.Error())
		}
		beers, err := bar.SearchForYummyAndWellPricedBeers()
		if err != nil {
			log.Printf("Searching for best priced beers error: %v", err)
			errors = append(errors, err.Error())
		}
		if len(errors) > 0 {
			beerInfo.Errors = errors
		}
		if len(beers) > 0 {
			beerInfo.Beers = beers
		}
		if len(beers) > 0 || len(errors) > 0 {
			barsWithGoodPrices = append(barsWithGoodPrices, beerInfo)
		}
	}

	return barsWithGoodPrices, nil
}

func (b Bar) SearchForYummyAndWellPricedBeers() ([]Beer, error) {
	var wellPricedBeers []Beer
	var searchErrors []error
	for _, beer := range *b.Beers {
		for _, priceStr := range strings.Split(beer.Prices, " · ") {
			if strings.Contains(priceStr, "0.5l:") {
				price, err := strconv.Atoi(strings.Replace(strings.Split(priceStr, ": ")[1], "zł", "", 1))
				if err != nil {
					searchErrors = append(searchErrors, err)
				}
				if price < PriceLimit {
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
