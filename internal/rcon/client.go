package rcon

import (
	"fmt"
	"net"

	gorcon "github.com/gorcon/rcon"
)

// Client wraps the gorcon RCON client.
type Client struct {
	conn *gorcon.Conn
}

// Connect opens an RCON connection to the given host and port.
func Connect(host string, port int, password string) (*Client, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := gorcon.Dial(addr, password)
	if err != nil {
		return nil, fmt.Errorf("rcon connect to %s: %w", addr, err)
	}
	return &Client{conn: conn}, nil
}

// Execute sends a command and returns the response.
func (c *Client) Execute(command string) (string, error) {
	resp, err := c.conn.Execute(command)
	if err != nil {
		return "", fmt.Errorf("rcon execute: %w", err)
	}
	return resp, nil
}

// Close closes the RCON connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
