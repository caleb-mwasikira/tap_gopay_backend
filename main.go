package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
)

var (
	address       string = "127.0.0.1:5000"
	migrate       bool
	mysqlUser     string
	mysqlPassword string
)

func init() {
	flag.StringVar(&address, "address", address, "IP address to run the server on")
	flag.BoolVar(&migrate, "migrate", false, "Run migration.")
	flag.StringVar(&mysqlUser, "mysql-user", "", "MySQL user to run migration")
	flag.StringVar(&mysqlPassword, "mysql-pass", "", "MySQL password to run migration")
	flag.Parse()

	if err := utils.ValidateAddress(address); err != nil {
		log.Fatalf("invalid IP address; %v\n", err)
	}
}

func main() {
	if migrate {
		err := database.MigrateDatabase(mysqlUser, mysqlPassword)
		if err != nil {
			log.Fatalf("Error migrating database; %v\n", err)
		}
		return
	}

	androidApiKey := api.GenerateAndroidApiKey()
	log.Println("ANDROID_API_KEY: ", androidApiKey)

	log.Printf("Starting web server on http://%v\n", address)

	routes := handlers.GetRoutes()
	err := http.ListenAndServe(address, routes)
	if err != nil {
		log.Fatalf("Error starting web server; %v\n", err)
	}
}
