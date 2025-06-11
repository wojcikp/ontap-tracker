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
	var e error

	c := ct.newCollector()

	c.OnError(func(_ *colly.Response, err error) {
		e = err
		log.Print("Something went wrong: ", err)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
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
		bars = append(bars, Bar{name, url, &[]Beer{}})
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println(r.Request.URL, " scraped!")
	})

	c.Visit("https://ontap.pl/warszawa/multitaps")

	if e != nil {
		return []Bar{}, e
	}
	return bars, nil
}

func (ct CollyTracker) FetchBeersInfo(wg *sync.WaitGroup, bar *Bar) error {
	defer wg.Done()
	var e error
	var name, prices string

	c := ct.newCollector()

	c.OnError(func(_ *colly.Response, err error) {
		e = err
		log.Print("Something went wrong: ", err)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
	})

	c.OnHTML("div.panel.panel-default", func(e *colly.HTMLElement) {
		e.DOM.Find("h4.cml_shadow").Each(func(_ int, s *goquery.Selection) {
			name = strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), "\t", "")
		})
		e.DOM.Find("div.col-xs-7").Each(func(_ int, s *goquery.Selection) {
			prices = strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), "\t", "")
		})
		*bar.Beers = append(*bar.Beers, Beer{name, prices})
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println(r.Request.URL, " scraped!")
	})

	c.Visit(bar.Url)

	return e
}
