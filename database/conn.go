package database

import (
	"database/sql"
	"log"
	"os"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
	"github.com/go-sql-driver/mysql"
)

var (
	db *sql.DB

	SECRET_KEY string
)

func init() {
	utils.LoadDotenv()

	SECRET_KEY = os.Getenv("SECRET_KEY")
	if strings.TrimSpace(SECRET_KEY) == "" {
		log.Fatalln("Missing env variable SECRET_KEY")
	}

	config := mysql.Config{
		User:                 os.Getenv("MYSQL_USER"),
		Passwd:               os.Getenv("MYSQL_PASSWORD"),
		DBName:               os.Getenv("MYSQL_DBNAME"),
		AllowNativePasswords: true,
		MultiStatements:      true,
		ParseTime:            true,
	}

	var err error
	db, err = openDbConn(config)
	if err != nil {
		log.Printf("Error opening database connection; %v\n", err)
		os.Exit(1)
	}

	// During first launch, our database is not going to have any tables
	// we need to create them immediately
	// TODO: migrate tables
}

func openDbConn(config mysql.Config) (*sql.DB, error) {
	log.Println("Opening database connection")

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, err
	}

	// Verify that database is actually open
	err = db.Ping()
	return db, err
}
