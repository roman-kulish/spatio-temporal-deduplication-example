package config

import "time"

const (
	defaultAddr = ":8080"
)

// Tolerance contains deduplication tolerance parameters.
type Tolerance struct {
	// Distance is a distance tolerance between location events in meters.
	Distance float64

	// Interval is a time tolerance between location events.
	Interval time.Duration
}

type Server struct {
	// Addr specifies the address for the server to listen on.
	Addr string

	// ReadTimeout is the maximum duration for reading the entire request,
	// including the body.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is the amount of time allowed to read request headers.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of
	// the response.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the next request
	// when keep-alive is enabled. If IdleTimeout is zero, the value of ReadTimeout
	// is used. If both are zero, there is no timeout.
	IdleTimeout time.Duration

	// ShutdownTimeout is a time to wait, until HTTP Server gracefully shutdowns.
	ShutdownTimeout time.Duration
}

// Config contains application configuration.
type Config struct {
	// DBPath is the path to the database directory.
	DBPath string

	Server
	Tolerance
}

// newConfig returns Config instance with default settings. The Config may not
// be usable yet.
func newConfig() *Config {
	return &Config{
		Server: Server{
			Addr: defaultAddr,
		},
	}
}
