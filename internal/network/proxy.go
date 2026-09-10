package network

import (
	"bufio"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// ProxyRotator rotates a file-backed proxy list using an atomic counter.
// Supported schemes: http, https, socks5 and socks5h.
type ProxyRotator struct {
	proxies []string
	index   atomic.Uint64
}

// NewProxyRotator loads proxies, ignoring blank lines and comments.
func NewProxyRotator(filePath string) (*ProxyRotator, error) {
	if filePath == "" {
		return &ProxyRotator{}, nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var list []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Assume HTTP when the scheme is omitted.
		if !strings.Contains(line, "://") {
			line = "http://" + line
		}
		list = append(list, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &ProxyRotator{proxies: list}, nil
}

// HasProxies reports whether the proxy list is nonempty.
func (pr *ProxyRotator) HasProxies() bool {
	return len(pr.proxies) > 0
}

// GetNext returns the next proxy safely across goroutines.
func (pr *ProxyRotator) GetNext() string {
	if len(pr.proxies) == 0 {
		return ""
	}
	idx := pr.index.Add(1) - 1 // 0-based
	return pr.proxies[idx%uint64(len(pr.proxies))]
}

// FetchProxies retrieves the public ProxyScrape list.
// It returns host:port strings with the scheme removed.
// Errors are logged and return an empty list.
func FetchProxies() []string {
	const apiURL = "https://api.proxyscrape.com/v4/free-proxy-list/get?request=display_proxies&proxy_format=protocolipport&format=text"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		slog.Error("fetch proxies failed", "error", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("read proxy response failed", "error", err)
		return nil
	}

	lines := strings.Split(string(body), "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// ProxyScrape returns scheme://host:port entries.
		// Retain only host:port.
		if idx := strings.Index(line, "://"); idx != -1 {
			line = line[idx+3:]
		}
		out = append(out, line)
	}
	return out
}
