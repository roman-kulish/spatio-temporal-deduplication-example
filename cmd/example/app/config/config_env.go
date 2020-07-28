package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	envDBPath                  = "DB_PATH"
	envDistanceTolerance       = "DISTANCE_TOLERANCE"
	envIntervalTolerance       = "INTERVAL_TOLERANCE"
	envServerAddr              = "SERVER_ADDR"
	envServerReadTimeout       = "SERVER_READ_TIMEOUT"
	envServerReadHeaderTimeout = "SERVER_READ_HEADER_TIMEOUT"
	envServerWriteTimeout      = "SERVER_WRITE_TIMEOUT"
	envServerShutdownTimeout   = "SERVER_SHUTDOWN_TIMEOUT"
	envServerIdleTimeout       = "SERVER_IDLE_TIMEOUT"
)

var envVars = []string{
	envDBPath,
	envDistanceTolerance,
	envIntervalTolerance,
	envServerAddr,
	envServerReadTimeout,
	envServerReadHeaderTimeout,
	envServerWriteTimeout,
	envServerIdleTimeout,
	envServerShutdownTimeout,
}

// NewFromEnv returns an instance of Config with default settings and
// configuration from environment variables.
func NewFromEnv() (*Config, error) {
	cfg := newConfig()

	for _, v := range envVars {
		val := os.Getenv(v)
		if val == "" {
			continue
		}

		var err error
		switch v {
		case envDBPath:
			cfg.DBPath = val
		case envDistanceTolerance:
			cfg.Tolerance.Distance, err = strconv.ParseFloat(val, 64)
		case envIntervalTolerance:
			cfg.Tolerance.Interval, err = time.ParseDuration(val)
		case envServerAddr:
			cfg.Server.Addr = val
		case envServerReadTimeout:
			cfg.Server.ReadTimeout, err = time.ParseDuration(val)
		case envServerReadHeaderTimeout:
			cfg.Server.ReadHeaderTimeout, err = time.ParseDuration(val)
		case envServerWriteTimeout:
			cfg.Server.WriteTimeout, err = time.ParseDuration(val)
		case envServerIdleTimeout:
			cfg.Server.IdleTimeout, err = time.ParseDuration(val)
		case envServerShutdownTimeout:
			cfg.Server.ShutdownTimeout, err = time.ParseDuration(val)
		}
		if err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
	}
	return cfg, nil
}
