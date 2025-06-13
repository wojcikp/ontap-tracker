package main

import (
	"github.com/wojcikp/ontap-tracker/internal/api"
)

func main() {
	server := api.NewServer("3000")
	server.Run()
}
