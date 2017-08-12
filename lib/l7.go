package lib

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
)

var (
	ErrInvalidAddress = errors.Errorf("Supplied address is invalid")
)

type L7 struct {
	sync.RWMutex

	logger         zerolog.Logger
	publicBackends map[string]Backend
	users          [][]byte
	port           int
	listener       net.Listener
	backends       map[string]*fasthttp.LBClient
}

func New(cfg Config) (lb L7, err error) {
	lb.port = cfg.Port

	if cfg.Debug {
		lb.logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		lb.logger = zerolog.New(os.Stderr)
	}

	if len(cfg.Users) > 0 {
		lb.LoadUsers(cfg.Users)
	}

	err = lb.LoadBackends(cfg.Backends)
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't load backends")
		return
	}

	return
}

func (lb *L7) LoadUsers(users map[string]string) {
	var ndx int = 0

	lb.logger.Debug().
		Int("users", len(users)).
		Msg("loading users")

	lb.users = make([][]byte, len(users))
	for login, pwd := range users {
		user := fmt.Sprintf("%s:%s", login, pwd)
		lb.users[ndx] = append(
			[]byte("Basic: "),
			base64.StdEncoding.EncodeToString([]byte(user))...)
		ndx++

		lb.logger.Debug().
			Str("user", "login").
			Msg("user loaded")
	}
}

func (lb *L7) LoadBackends(backends map[string]Backend) (err error) {
	var (
		lbc  *fasthttp.LBClient
		be   Backend
		url  string
		name string

		internalBackends = make(map[string]*fasthttp.LBClient)
	)

	lb.logger.Debug().
		Int("total", len(backends)).
		Msg("loading backends")

	for name, be = range backends {
		if len(be.Servers) == 0 {
			lb.logger.Debug().Str("backend", name).Msg("no servers")
			internalBackends[name] = nil
			continue
		}

		lb.logger.Debug().
			Str("backend", name).
			Int("total", len(be.Servers)).
			Msg("loading servers")

		lbc = &fasthttp.LBClient{}
		for _, server := range be.Servers {
			url, err = NormalizeAddress(server.Address)
			if err != nil {
				err = errors.Wrapf(err,
					"Can't use address %s as a server address",
					server.Address)
				return
			}

			lb.logger.Debug().
				Str("backend", name).
				Str("server", url).
				Msg("server loaded")

			lbc.Clients = append(lbc.Clients, &fasthttp.HostClient{
				Addr: url,
			})
		}

		lb.logger.Debug().
			Str("backend", name).
			Msg("backend loaded")
		internalBackends[name] = lbc
	}

	lb.publicBackends = backends

	lb.Lock()
	defer lb.Unlock()
	lb.backends = internalBackends
	return
}

func (lb *L7) GetBackends() map[string]Backend {
	lb.RLock()
	defer lb.RUnlock()

	return lb.publicBackends
}

var (
	authorizationHeader = []byte("Authorization")
	authenticateHeader  = []byte("WWW-Authenticate")
	authenticateRealm   = []byte("Basic realm=\"basic\"")
	connectionHeader    = []byte("Connection")
)

func (lb *L7) authenticate(ctx *fasthttp.RequestCtx) (ok bool) {
	var (
		auth []byte
	)

	auth = ctx.Request.Header.PeekBytes(authorizationHeader)
	if len(auth) == 0 {
		ctx.Response.Header.SetBytesKV(
			authenticateHeader, authenticateRealm)
		ctx.SetStatusCode(401)
		return
	}

	for _, usr := range lb.users {
		if bytes.Equal(auth, usr) {
			ok = true
			return
		}
	}
	return
}

func (lb *L7) route(ctx *fasthttp.RequestCtx) {
	var (
		ndx int
		b   byte
	)

	for ndx, b = range ctx.Host() {
		if b == ':' {
			ndx--
			break
		}
	}
	ndx++

	lb.logger.Info().Bytes("host", ctx.Host()[:ndx]).Msg("routing")

	lb.RLock()
	backend, found := lb.backends[string(ctx.Host()[:ndx])]
	lb.RUnlock()
	if !found {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}
	if backend == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}

	ctx.Request.Header.DelBytes(connectionHeader)
	err := backend.Do(&ctx.Request, &ctx.Response)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
	}
	ctx.Response.Header.DelBytes(connectionHeader)
}

func (lb *L7) handler(ctx *fasthttp.RequestCtx) {
	if len(lb.users) > 0 {
		if !lb.authenticate(ctx) {
			lb.logger.Info().Msg("not authenticated")
			return
		}
	}

	lb.route(ctx)
}

func (lb *L7) Listen() (err error) {
	ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", lb.port))
	if err != nil {
		err = errors.Wrapf(err,
			"couldn't listen on port %d",
			lb.port)
		return
	}

	lb.port = ln.Addr().(*net.TCPAddr).Port
	lb.listener = ln

	err = (&fasthttp.Server{
		Name: "cirocosta/l7",
		DisableHeaderNamesNormalizing: true,
		Handler: lb.handler,
	}).Serve(ln)
	if err != nil {
		err = errors.Wrapf(err,
			"couldn't serve http handler")
		return
	}

	return
}

// TODO implement gracefull shutdown
func (lb *L7) Stop() {
	if lb.listener != nil {
		lb.listener.Close()
	}
}
