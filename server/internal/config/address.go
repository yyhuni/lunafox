package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

// DSN returns the database connection string.
func (c *DatabaseConfig) DSN() string {
	dsn := &url.URL{
		Scheme:  "postgresql",
		User:    url.UserPassword(c.User, c.Password),
		Host:    net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:    "/" + c.Name,
		RawPath: "/" + url.PathEscape(c.Name),
	}
	query := dsn.Query()
	query.Set("sslmode", c.SSLMode)
	query.Set("TimeZone", c.TimeZone)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

// Addr returns the Redis address.
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
