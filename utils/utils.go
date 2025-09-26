package utils

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	projectDir string
	SecretsDir string
)

func init() {
	_, file, _, _ := runtime.Caller(0)
	utilsDir := filepath.Dir(file)
	projectDir = filepath.Dir(utilsDir)
	SecretsDir = filepath.Join(projectDir, "secrets")
}

func trimComments(s string) string {
	comment := "#"
	sb := []string{}

	for _, char := range s {
		if string(char) == comment {
			return strings.Join(sb, "")
		}
		sb = append(sb, string(char))
	}
	return strings.Join(sb, "")
}

var once sync.Once

func LoadDotenv() {
	// Makes sure function is only called once
	once.Do(func() {
		log.Println("Loading env variables")
		envFile := filepath.Join(projectDir, ".env")
		data, err := os.ReadFile(envFile)
		if err != nil {
			log.Fatalf("Error loading env variables; %v\n", err)
		}

		lines := strings.Split(string(data), "\n")

		for _, line := range lines {
			line = trimComments(line)
			line = strings.TrimSpace(line)

			if line == "" {
				continue
			}

			fields := strings.Split(line, "=")
			if len(fields) != 2 {
				continue
			}

			key := fields[0]
			value := fields[1]
			value = strings.Trim(value, "\"")
			os.Setenv(key, value)
		}
	})
}

func ValidateAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}

	_, err = net.ResolveIPAddr("ip", host)
	if err != nil {
		return err
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	if portNum < 1 || portNum >= math.MaxUint16 {
		return fmt.Errorf("port number is not within range 1 - %v", math.MaxUint16)
	}
	return nil
}

// Returns elements from slice where predicate is true and
// elements from slice where predicate is false.
// If predicate == nil returns 2 empty slices
func Filter[T any](items []T, predicate func(item T) bool) ([]T, []T) {
	meetsPredicate := []T{}
	doesntMeetPredicate := []T{}

	if predicate == nil {
		return meetsPredicate, doesntMeetPredicate
	}

	for _, item := range items {
		if predicate(item) {
			meetsPredicate = append(meetsPredicate, item)
		} else {
			doesntMeetPredicate = append(doesntMeetPredicate, item)
		}
	}
	return meetsPredicate, doesntMeetPredicate
}

// roundFloat rounds a float64 to a specified number of decimal places.
func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func FindOne[T any](items []T, predicate func(item T) bool) *T {
	if predicate == nil {
		return nil
	}

	for _, item := range items {
		if predicate(item) {
			return &item
		}
	}
	return nil
}
