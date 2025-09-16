package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-sql-driver/mysql"
)

func Unauthorized(w http.ResponseWriter, message string, args ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	message = fmt.Sprintf(message, args...)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// Returns 401 Unauthorized response to the user.
//
//	parameter message is displayed to the user
//	parameter err is logged if err != nil
func BadRequest(w http.ResponseWriter, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	if err != nil {
		log.Printf("%v %v; %v\n",
			http.StatusText(http.StatusBadRequest), message, err)
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// Returns a 500 InternalServerError response to user
func Errorf(w http.ResponseWriter, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	log.Printf("%v %v; %v\n", http.StatusInternalServerError, message, err)

	// MySQL error code 1644 ER_SIGNAL_EXCEPTION - Unhandled user-defined exception condition
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		if mysqlErr.Number == 1644 {
			message += fmt.Sprintf(". %v", mysqlErr.Message)
		}
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// Sends 200 OK response to the user with message as response body.
// The response body is serialized into the JSON format;
//
//	{
//		"message": <Your message goes here>
//	}
func OK(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// Sends 200 OK response to the user with data as response body
func OK2(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// Sends 409 Conflict response to the user
func Conflict(w http.ResponseWriter, format string, args ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)

	message := fmt.Sprintf(format, args...)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// Sends 404 NotFound response to the user
func NotFound(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)

	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}
