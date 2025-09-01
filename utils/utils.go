package utils

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
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
