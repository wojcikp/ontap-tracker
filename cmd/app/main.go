package main

import (
	"github.com/wojcikp/ontap-tracker/internal/tracker"
)

func main() {
	ontapScrapper := tracker.NewCollyTracker()
	bars, _ := ontapScrapper.FetchBarsInWarsaw()
	for _, bar := range bars[len(bars)-3:] {
		ontapScrapper.FetchBeersInfo(&bar)
	}

}
