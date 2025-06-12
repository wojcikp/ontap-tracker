package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/wojcikp/ontap-tracker/internal/tracker"
)

func main() {
	r := mux.NewRouter()
	h := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:8080"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type", "application/json"}),
	)(r)

	r.HandleFunc("/scrap", scrapHandler).Methods("GET")

	http.ListenAndServe(":3000", h)

}

func scrapHandler(w http.ResponseWriter, r *http.Request) {

	ontapScrapper := tracker.NewCollyTracker()
	bars, err := ontapScrapper.FetchBarsInWarsaw()
	if err != nil {
		log.Fatal("Fetching bars error: ", err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(bars))
	// wg.Add(3)
	for _, bar := range bars {
		// for _, bar := range bars[len(bars)-3:] {
		go ontapScrapper.FetchBeersInfo(wg, &bar)
	}

	wg.Wait()

	for _, bar := range bars {
		if bar.ScrapeErr != nil {
			log.Printf("Scrape error in bar: %s \nERROR: %v", bar.Name, bar.ScrapeErr)
			continue
		}
		beers, err := bar.SearchForYummyAndWellPricedBeers()
		if err != nil {
			log.Printf("Searching for best priced beers error: %v", err)
		}
		if len(beers) > 0 {
			// w.Write([]byte(bar.Name))
			// for _, beer := range beers {
			// 	w.Write([]byte(fmt.Sprintf("%v", beer)))
			// }
			json.NewEncoder(w).Encode(bar.Name)
			json.NewEncoder(w).Encode(beers)
		}
	}
}
