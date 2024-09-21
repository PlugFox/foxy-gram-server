package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/config"
	"golang.org/x/net/proxy"
)

func NewHttpClient(config *config.ProxyConfig) (*http.Client, error) {
	transport := &http.Transport{}
	if config != nil && config.Address != "" && config.Port != 0 {
		addr := fmt.Sprintf("%s:%s", config.Address, strconv.Itoa(config.Port))
		var auth *proxy.Auth
		if config.Username != "" && config.Password != "" {
			auth = &proxy.Auth{User: config.Username, Password: config.Password}
		}
		dialer, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("cannot init socks5 proxy client dialer: %w", err)
		}
		transport.DialContext = func(_ context.Context, network, address string) (net.Conn, error) {
			return dialer.Dial(network, address)
		}
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second, // Set a timeout to avoid hanging requests
	}
	return httpClient, nil
}
