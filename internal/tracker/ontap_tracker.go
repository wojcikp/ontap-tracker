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
	Name         string  `json:"bar"`
	Address      string  `json:"adres"`
	Url          string  `json:"url"`
	Beers        []Beer  `json:"piwa"`
	ScrapeErrors []error `json:"errors"`
}

type Beer struct {
	Name              string
	Prices            string
	PriceForHalfLiter int
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
		beers, err := bar.SearchForWellPricedBeers(priceLimit)
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

func (ct CollyTracker) GetBeerWithLowestPrice() ([]Beer, error) {
	bars := make(chan Bar)
	barsUrls, err := ct.FetchBarsUrls()
	if err != nil {
		log.Print("ERROR during fetching bars urls: ", err)
		return []Beer{}, err
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

	var allBeers []Beer
	for bar := range bars {
		allBeers = append(allBeers, bar.Beers...)
	}

	cheapestBeer, err := SearchForBeerWithLowestPrice(allBeers)
	if err != nil {
		return []Beer{}, fmt.Errorf("error occured during searching for lowest price beer: %w", err)
	}

	var cheapestBeersInBars []Beer
	cheapestBeersInBars = append(cheapestBeersInBars, cheapestBeer)

	for _, beer := range allBeers {
		if beer.PriceForHalfLiter == cheapestBeer.PriceForHalfLiter && beer != cheapestBeer {
			cheapestBeersInBars = append(cheapestBeersInBars, beer)
		}
	}

	return cheapestBeersInBars, nil
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
	var scrapeErrors []error

	c := ct.newCollector()

	c.OnError(func(_ *colly.Response, err error) {
		scrapeErrors = append(scrapeErrors, err)
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
		priceForHalfLiter, err := GetBeerPrice(prices)
		if err != nil {
			scrapeErrors = append(scrapeErrors, err)
		}
		beers = append(beers, Beer{
			Name:              beerName,
			Prices:            prices,
			PriceForHalfLiter: priceForHalfLiter,
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
		bar <- Bar{name, address, barUrl, beers, scrapeErrors}
	})

	if err := c.Visit(barUrl); err != nil {
		scrapeErrors = append(scrapeErrors, err)
	}
}

func (b Bar) SearchForWellPricedBeers(priceLimit int) ([]Beer, error) {
	var wellPricedBeers []Beer
	var searchErrors []error
	for _, beer := range b.Beers {
		if beer.PriceForHalfLiter < priceLimit && beer.PriceForHalfLiter > 0 {
			wellPricedBeers = append(wellPricedBeers, beer)
		}
	}
	if len(searchErrors) > 0 {
		return nil, fmt.Errorf("errors occured during search for well priced beers: %v", searchErrors)
	}
	return wellPricedBeers, nil
}

func SearchForBeerWithLowestPrice(beers []Beer) (Beer, error) {
	lowestPriceBeer := beers[0]
	var searchErrors []error
	for _, beer := range beers {
		if beer.PriceForHalfLiter < lowestPriceBeer.PriceForHalfLiter && beer.PriceForHalfLiter > 0 {
			lowestPriceBeer = beer
		}
	}
	if len(searchErrors) > 0 {
		return Beer{}, fmt.Errorf("errors occured during search for lowest priced beers: %v", searchErrors)
	}
	return lowestPriceBeer, nil
}

func GetBeerPrice(prices string) (int, error) {
	var price int
	var err error
	for _, priceStr := range strings.Split(prices, " · ") {
		if strings.Contains(priceStr, "0.5l:") {
			price, err = strconv.Atoi(strings.Replace(strings.Split(priceStr, ": ")[1], "zł", "", 1))
			if err != nil {
				return price, err
			}
		}
	}
	return price, nil
}
