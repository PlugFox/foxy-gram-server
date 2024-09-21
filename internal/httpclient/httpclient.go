package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/plugfox/foxy-gram-server/internal/config"
	"golang.org/x/net/proxy"
)

func NewHttpSocks5Client(config *config.ProxyConfig) (*http.Client, error) {
	if config == nil || config.Address == "" || config.Port == 0 {
		return nil, nil
	}
	addr := fmt.Sprintf("%s:%s", config.Address, strconv.Itoa(config.Port))
	var auth *proxy.Auth
	if config.Username != "" && config.Password != "" {
		auth = &proxy.Auth{User: config.Username, Password: config.Password}
	}
	dialer, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("cannot init socks5 proxy client dialer: %w", err)
	}
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}
	httpTransport.DialContext = func(_ context.Context, network, address string) (net.Conn, error) {
		return dialer.Dial(network, address)
	}
	return httpClient, nil
}
