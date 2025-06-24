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
	tracker := tracker.NewCollyTracker()
	beersInfo, err := tracker.GetBeersInfo(numericalPriceLimit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
	}
	if len(beersInfo) > 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(beersInfo)
	} else {
		w.WriteHeader(http.StatusNoContent)
		json.NewEncoder(w).Encode("No well priced beers found")
	}
}
