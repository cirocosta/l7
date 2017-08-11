package lib

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const (
	HTTP_SCHEMA_PREFIX  = "http://"
	HTTPS_SCHEMA_PREFIX = "https://"
)

func NormalizeAddress(address string) (res string, err error) {
	var (
		port string
		host string
	)

	if !strings.HasPrefix(address, HTTP_SCHEMA_PREFIX) &&
		!strings.HasPrefix(address, HTTPS_SCHEMA_PREFIX) {
		address = HTTP_SCHEMA_PREFIX + address
	}

	parsedUrl, err := url.Parse(address)
	if err != nil {
		err = errors.Wrapf(err, "Invalid address %s", address)
		return
	}

	if parsedUrl.Hostname() == "" {
		err = errors.Errorf("Address does not have a host")
		return
	}

	host = parsedUrl.Hostname()
	port = parsedUrl.Port()

	if port == "" {
		switch parsedUrl.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}

	res = host + ":" + port
	return
}
