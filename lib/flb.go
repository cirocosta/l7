package lib

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

var (
	ErrInvalidAddress = errors.Errorf("Supplied address is invalid")
)

type Flb struct {
	sync.RWMutex

	publicBackends map[string]Backend
	port           int
	listener       net.Listener
	backends       map[string]*fasthttp.LBClient
}

func New(cfg Config) (lb Flb, err error) {
	lb.port = cfg.Port

	err = lb.LoadBackends(cfg.Backends)
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't load backends")
		return
	}

	return
}

func (lb *Flb) LoadBackends(backends map[string]Backend) (err error) {
	var internalBackends = make(map[string]*fasthttp.LBClient)
	var lbc *fasthttp.LBClient
	var url string

	for name, be := range backends {
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

func (lb *Flb) GetBackends() map[string]Backend {
	lb.RLock()
	defer lb.RUnlock()

	return lb.publicBackends
}

func (lb *Flb) handler(ctx *fasthttp.RequestCtx) {
	var host = strings.Split(string(ctx.Host()), ":")[0]

	lb.RLock()
	backend, found := lb.backends[host]
	lb.RUnlock()
	if !found {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}
	if backend == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}

	req := &ctx.Request
	resp := &ctx.Response
	req.Header.Del("Connection")

	err := backend.Do(req, resp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
	}
	resp.Header.Del("Connection")

}

func (lb *Flb) Listen() (err error) {
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
func (lb *Flb) Stop() {
	if lb.listener != nil {
		lb.listener.Close()
	}
}
