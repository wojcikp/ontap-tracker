package tracker

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

type Tracker interface {
	FetchBarsInWarsaw() []Bar
}

type CollyTracker struct{}

type Bar struct {
	Name  string
	Url   string
	Beers *[]Beer
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

	c.OnScraped(func(r *colly.Response) {
		fmt.Println(r.Request.URL, " scraped!")
	})

	if err := c.Visit("https://ontap.pl/warszawa/multitaps"); err != nil {
		return nil, err
	}

	if scrapeErr != nil {
		return nil, scrapeErr
	}
	return bars, nil
}

func (ct CollyTracker) FetchBeersInfo(wg *sync.WaitGroup, bar *Bar) error {
	defer wg.Done()

	var beers []Beer
	var scrapeErr error

	c := ct.newCollector()

	c.OnError(func(_ *colly.Response, err error) {
		scrapeErr = err
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
		fmt.Println(r.Request.URL, " scraped!")
	})

	if err := c.Visit(bar.Url); err != nil {
		return err
	}

	*bar.Beers = beers

	return scrapeErr
}
