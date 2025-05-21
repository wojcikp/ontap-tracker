package tracker

import (
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

type Tracker interface {
	FetchBarsInWarsaw() []Bar
}

type CollyTracker struct {
	colly *colly.Collector
}

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
	c := colly.NewCollector(colly.AllowedDomains())
	return &CollyTracker{c}
}

func (ct CollyTracker) FetchBarsInWarsaw() ([]Bar, error) {
	var bars []Bar
	var e error

	ct.colly.OnError(func(_ *colly.Response, err error) {
		e = err
		log.Print("Something went wrong: ", err)
	})

	ct.colly.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
	})

	ct.colly.OnHTML("div.panel.panel-default.text-center", func(e *colly.HTMLElement) {
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

	ct.colly.OnScraped(func(r *colly.Response) {
		fmt.Println(r.Request.URL, " scraped!")
	})

	ct.colly.Visit("https://ontap.pl/warszawa/multitaps")

	if e != nil {
		return []Bar{}, e
	}
	return bars, nil
}

func (ct CollyTracker) FetchBeersInfo(bar *Bar) error {
	var e error
	var name, prices string

	ct.colly.OnError(func(_ *colly.Response, err error) {
		e = err
		log.Print("Something went wrong: ", err)
	})

	ct.colly.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
	})

	ct.colly.OnHTML("div.panel.panel-default", func(e *colly.HTMLElement) {
		e.DOM.Find("h4.cml_shadow").Each(func(_ int, s *goquery.Selection) {
			name = strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), "\t", "")
		})
		e.DOM.Find("div.col-xs-7").Each(func(_ int, s *goquery.Selection) {
			prices = strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), "\t", "")
		})
		*bar.Beers = append(*bar.Beers, Beer{name, prices})
	})

	ct.colly.OnScraped(func(r *colly.Response) {
		fmt.Println(r.Request.URL, " scraped!")
	})

	ct.colly.Visit(bar.Url)

	return e
}
