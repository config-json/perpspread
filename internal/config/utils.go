package config

import (
	"os"
	"strconv"
)

func getEnv(key string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	panic("missing environment variable:" + key)
}

func getEnvInt(key string) int {
	v := getEnv(key)
	i, err := strconv.Atoi(v)

	if err != nil {
		panic("invalid integer environment variable:" + key)
	}

	return i
}
