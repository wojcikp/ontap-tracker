package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/wojcikp/ontap-tracker/internal/tracker"
)

const DefaultPriceLimit = "18"

type Server struct {
	Port   string
	Router *mux.Router
}

func NewServer(port string) *Server {
	r := mux.NewRouter()
	return &Server{Port: port, Router: r}
}

func (s *Server) Run() {
	h := handlers.CORS(
		// handlers.AllowedOrigins([]string{"http://localhost:8080"}),
		handlers.AllowedMethods([]string{"GET"}),
		handlers.AllowedHeaders([]string{"Content-Type", "application/json"}),
	)(s.Router)

	s.Router.HandleFunc("/beers", s.scrapHandler).Methods("GET")
	s.Router.HandleFunc("/lowest", s.lowestPriceHandler).Methods("GET")

	p := fmt.Sprint(":", s.Port)
	log.Printf("Server running on http://localhost%s", p)
	http.ListenAndServe(p, h)
}

func (s *Server) scrapHandler(w http.ResponseWriter, r *http.Request) {
	priceLimit := r.URL.Query().Get("price")
	if priceLimit == "" {
		priceLimit = DefaultPriceLimit
	}
	numericalPriceLimit, err := strconv.Atoi(priceLimit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
	}
	t := tracker.NewCollyTracker()
	bars, err := t.GetBarsWithWellPricedBeers(numericalPriceLimit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bars)
}

func (s *Server) lowestPriceHandler(w http.ResponseWriter, r *http.Request) {
	t := tracker.NewCollyTracker()
	b, err := t.GetBeerWithLowestPrice()
	if err != nil {
		log.Print(err)
		json.NewEncoder(w).Encode(err)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(b)
}
