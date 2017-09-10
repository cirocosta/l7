package lib

import (
	"bytes"
	"sync"

	"github.com/rs/zerolog"
)

type Authenticator struct {
	users      [][]byte
	ipCounters sync.Map
	logger zerolog.Logger
}

func (a *Authenticator) Authenticate(header string) (ok bool, err error) {
	if header == "" {
		a.logger.Info().
			Msg("auth header required but not present")
	}

	for _, usr := range a.users {
		a.logger.Info().
			Bytes("usr", usr).
			Bytes("auth", auth).
			Msg("checking")

		if bytes.Equal(auth, usr) {
			lb.logger.Debug().
				Msg("authentication succeeded")
			ok = true
			return
		}
	}

	lb.logger.Info().
		Msg("no allowed user found")
	return
}

func (a *Authenticator) TryRateLimitedAuthenticate(header, ip string) (ok bool, err error) {

}
