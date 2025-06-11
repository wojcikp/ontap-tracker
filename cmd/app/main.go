package main

import (
	"log"
	"sync"

	"github.com/wojcikp/ontap-tracker/internal/tracker"
)

func main() {
	ontapScrapper := tracker.NewCollyTracker()
	bars, _ := ontapScrapper.FetchBarsInWarsaw()
	wg := &sync.WaitGroup{}
	wg.Add(3)
	for _, bar := range bars[len(bars)-3:] {
		ontapScrapper.FetchBeersInfo(wg, &bar)
	}

	wg.Wait()

	for _, bar := range bars {
		log.Print(bar.Name)
		log.Print(bar.Beers)
	}

}
