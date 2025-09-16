package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"slices"
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

//go:embed sql/*
var sqlDir embed.FS

// Requires root privileges
func MigrateDatabase(rootUser, rootPassword string) error {
	utils.LoadDotenv()

	// Create database connection
	config := mysql.Config{
		User:                 rootUser,
		Passwd:               rootPassword,
		DBName:               os.Getenv("MYSQL_DBNAME"),
		AllowNativePasswords: true,
		MultiStatements:      true,
		ParseTime:            true,
	}
	dbConn, err := openDbConn(config)
	if err != nil {
		return err
	}

	files, err := sqlDir.ReadDir("sql")
	if err != nil {
		return err
	}

	// Get users permission to truncate database
	fmt.Println("migrateTables function is going to TRUNCATE entire database. This could lead to data loss")
	fmt.Printf("Do you wish to continue? Y/n? ")

	var answer string
	if _, err = fmt.Scanln(&answer); err != nil {
		return err
	}

	if answer != "Y" {
		log.Println("migrateTables func aborted. Reason? Cancelled by user")
		return nil
	}

	query := "DROP DATABASE " + config.DBName
	if _, err = dbConn.Exec(query); err != nil {
		return err
	}

	query = "CREATE DATABASE " + config.DBName
	if _, err = dbConn.Exec(query); err != nil {
		return err
	}

	query = "USE " + config.DBName
	if _, err = dbConn.Exec(query); err != nil {
		return err
	}

	// Execute these files last as they contain FOREIGN KEY constraints
	last := []string{"extra.sql", "routines.sql"}

	lastFiles, files := utils.Filter(files, func(file fs.DirEntry) bool {
		placeLast := strings.HasPrefix(file.Name(), "view_") || slices.Contains(last, file.Name())
		return placeLast
	})

	files = append(files, lastFiles...)

	for _, file := range files {
		sqlFile := strings.HasSuffix(file.Name(), ".sql")
		if !sqlFile {
			continue
		}

		filename := fmt.Sprintf("sql/%v", file.Name())
		data, err := sqlDir.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("error reading sql file; %v", err)
		}

		log.Printf("Executing file '%v'\n", filename)

		query := string(data)
		_, err = dbConn.Exec(query)
		if err != nil {
			return fmt.Errorf("error executing query:\n`%v`\n on file %v\n%v", query, file.Name(), err)
		}
	}

	return nil
}
