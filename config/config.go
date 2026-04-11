package config

import "os"

type Config struct {
    RedisAddr  string
    ServerPort string
}

func Load() *Config {
    redisAddr := os.Getenv("REDIS_ADDR")
    if redisAddr == "" {
        redisAddr = "localhost:6379" // default
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    return &Config{
        RedisAddr:  redisAddr,
        ServerPort: port,
    }
}