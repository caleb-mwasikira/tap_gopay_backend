package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Returns 401 Unauthorized response to the user
func Unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "You are not authorized to access this resource",
	})
}

func Unauthorized2(w http.ResponseWriter, message string, args ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	message = fmt.Sprintf(message, args...)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// Returns 400 BadRequest response to the user
func BadRequest(w http.ResponseWriter, format string, args ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	message := fmt.Sprintf(format, args...)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// Returns 400 BadRequest response to the user.
// Use this when you want to returns validation errors to the
// user as and array/obj
func BadRequest2(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(data)
}

// Returns a 500 InternalServerError response to user
func Errorf(w http.ResponseWriter, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	status := http.StatusText(http.StatusInternalServerError)
	log.Printf("%v %v; %v\n", status, message, err)

	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// Sends 200 OK response to the user with message as response body.
// The message is serialized into the JSON format;
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
