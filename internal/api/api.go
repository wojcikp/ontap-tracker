package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/wojcikp/ontap-tracker/internal/tracker"
)

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
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type", "application/json"}),
	)(s.Router)

	s.Router.HandleFunc("/scrap", s.scrapHandler).Methods("GET")

	p := fmt.Sprint(":", s.Port)
	log.Printf("Server running on http://localhost%s", p)
	http.ListenAndServe(p, h)
}

func (s *Server) scrapHandler(w http.ResponseWriter, r *http.Request) {
	tracker := tracker.NewCollyTracker()
	beersInfo, err := tracker.GetBeersInfo()
	if err != nil {
		json.NewEncoder(w).Encode(err)
	}
	json.NewEncoder(w).Encode(beersInfo)
}
