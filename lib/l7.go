package lib

import (
	"fmt"
	"net"
	"sync"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

var (
	ErrInvalidAddress = errors.Errorf("Supplied address is invalid")
)

type L7 struct {
	sync.RWMutex

	publicBackends map[string]Backend
	users          map[string]string
	port           int
	listener       net.Listener
	backends       map[string]*fasthttp.LBClient
}

func New(cfg Config) (lb L7, err error) {
	lb.port = cfg.Port
	lb.users = cfg.Users

	err = lb.LoadBackends(cfg.Backends)
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't load backends")
		return
	}

	return
}

func (lb *L7) LoadBackends(backends map[string]Backend) (err error) {
	var (
		internalBackends = make(map[string]*fasthttp.LBClient)
		lbc              *fasthttp.LBClient
		be               Backend
		url              string
		name             string
	)

	for name, be = range backends {
		if len(be.Servers) == 0 {
			internalBackends[name] = nil
			continue
		}

		lbc = &fasthttp.LBClient{}
		for _, server := range be.Servers {
			url, err = NormalizeAddress(server.Address)
			if err != nil {
				err = errors.Wrapf(err,
					"Can't use address %s as a server address",
					server.Address)
				return
			}

			lbc.Clients = append(lbc.Clients, &fasthttp.HostClient{
				Addr: url,
			})
		}

		internalBackends[name] = lbc
	}

	lb.Lock()
	defer lb.Unlock()
	lb.publicBackends = backends
	lb.backends = internalBackends
	return
}

func (lb *L7) GetBackends() map[string]Backend {
	lb.RLock()
	defer lb.RUnlock()

	return lb.publicBackends
}

func (lb *L7) handler(ctx *fasthttp.RequestCtx) {
	var (
		ndx int
		b   byte
	)

	for ndx, b = range ctx.Host() {
		if b == ':' {
			break
		}
	}

	lb.RLock()
	backend, found := lb.backends[string(ctx.Host()[:ndx+1])]
	lb.RUnlock()
	if !found {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}
	if backend == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}

	ctx.Request.Header.Del("Connection")
	err := backend.Do(&ctx.Request, &ctx.Response)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
	}
	ctx.Response.Header.Del("Connection")

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

	err = fasthttp.Serve(lb.listener, lb.handler)
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
