package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port string
	RedisAddr string
	RedisPass string
	RateLimit int
	RateWindow time.Duration
	ReadTimeout time.Duration
	WriteTimeout time.Duration
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT","8080"),
		RedisAddr: getEnv("REDIS_ADDR","localhost:6379"),
		RedisPass: getEnv("REDIS_PASSWORD",""),
		RateLimit: getEnvAsInt("RATE_LIMIT",5),
		RateWindow:getEnvAsDuration("RATE_WINDOW_SEC",10*time.Second),
		ReadTimeout: 5* time.Second,
		WriteTimeout: 10* time.Second,

	}
}

func getEnv(key, defaulVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaulVal

}

func getEnvAsInt(key string, defaultVal int) int {
	var valStr = getEnv(key,"") 
	if val,err := strconv.Atoi(valStr); err != nil {
		return val
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(key,"")
	if val,err := strconv.Atoi(valStr); err == nil {
		return time.Duration(val) * time.Second
	}
	return defaultVal
}