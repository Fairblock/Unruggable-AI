package utils

import (
	"log"
	"os"
)


func GetEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("Missing required variable: %s", key)
	}
	return value
}