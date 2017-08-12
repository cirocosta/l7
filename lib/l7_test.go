package lib

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func createServer(name string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", name)
	}))
}

func targetHost(host string, port int) (resp *http.Response, err error) {
	var addr = fmt.Sprintf("http://localhost:%d", port)
	var client = &http.Client{}

	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return
	}

	req.Host = host
	resp, err = client.Do(req)
	return
}


func TestNew_doesntFailIfNoPortSpecified(t *testing.T) {
	_, err := New(Config{})
	assert.NoError(t, err)
}

func Test_404sServerNotRegistered(t *testing.T) {
	lb, err := New(Config{})
	assert.NoError(t, err)

	defer lb.Stop()

	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := targetHost("something.com", lb.port)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func Test_forwardsRequestsToServer(t *testing.T) {
	var name = RandomLowercaseString(5) + ".com"
	var server = createServer(name)
	defer server.Close()

	var servers = []Server{
		{
			Address: server.URL,
		},
	}

	lb, err := New(Config{
		Backends: map[string]Backend{
			name: Backend{
				Servers: servers,
			},
		},
	})
	assert.NoError(t, err)
	defer lb.Stop()
	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := targetHost(name, lb.port)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	data, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, name, string(data))
}

func TestNew_doesntfailIfNoServerSpecified(t *testing.T) {
	_, err := New(Config{
		Backends: map[string]Backend{
			"something.com": Backend{},
		},
	})
	assert.NoError(t, err)
}

func Test_respondsWith503IfBackendWithoutServers(t *testing.T) {
	lb, err := New(Config{
		Backends: map[string]Backend{
			"something.com": Backend{},
		},
	})
	assert.NoError(t, err)

	defer lb.Stop()
	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 10; i++ {
		resp, err := targetHost("something.com", lb.port)
		assert.NoError(t, err)
		assert.Equal(t, 503, resp.StatusCode)
	}
}

func Test_respondsWith502OnInvalidUpstream(t *testing.T) {
	var servers = []Server{
		{
			Address: "127.0.0.5:1337",
		},
	}

	lb, err := New(Config{
		Backends: map[string]Backend{
			"something.com": Backend{
				Servers: servers,
			},
		},
	})
	assert.NoError(t, err)

	defer lb.Stop()
	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 10; i++ {
		resp, err := targetHost("something.com", lb.port)
		assert.NoError(t, err)
		assert.Equal(t, 502, resp.StatusCode)
	}
}

func Test_withMultipleServers_balancesRequests(t *testing.T) {
	var name1 = RandomLowercaseString(5) + ".com"
	var name2 = RandomLowercaseString(5) + ".com"
	var name3 = RandomLowercaseString(5) + ".com"

	var server1 = createServer(name1)
	var server2 = createServer(name2)
	var server3 = createServer(name3)

	defer server1.Close()
	defer server2.Close()
	defer server3.Close()

	var servers = []Server{
		{
			Address: server1.URL,
		},
		{
			Address: server2.URL,
		},
		{
			Address: server3.URL,
		},
	}

	lb, err := New(Config{
		Backends: map[string]Backend{
			"something.com": Backend{
				Servers: servers,
			},
		},
	})
	assert.NoError(t, err)
	defer lb.Stop()
	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	var bodies = []string{}

	for i := 0; i < 10; i++ {
		resp, err := targetHost("something.com", lb.port)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		data, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)
		bodies = append(bodies, string(data))
	}

	assert.Contains(t, bodies, name1)
	assert.Contains(t, bodies, name2)
	assert.Contains(t, bodies, name3)
}

func Test_configurationReloading(t *testing.T) {
	var bodies = []string{}
	var name1 = "server1.com"
	var name2 = "server2.com"
	var server1 = createServer(name1)
	var server2 = createServer(name2)

	defer server1.Close()
	defer server2.Close()

	var backends = map[string]Backend{
		"something.com": Backend{
			Servers: []Server{
				{server1.URL},
			},
		},
	}

	lb, err := New(Config{Backends: backends})
	assert.NoError(t, err)

	defer lb.Stop()
	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 100; i++ {
		resp, err := targetHost("something.com", lb.port)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		if i == 50 {
			err = lb.LoadBackends(map[string]Backend{
				"something.com": Backend{
					Servers: []Server{
						{server2.URL},
					},
				},
			})
			assert.NoError(t, err)
		}

		data, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)
		bodies = append(bodies, string(data))
	}

	assert.Contains(t, bodies, name1)
	assert.Contains(t, bodies, name2)
}

func Test_reconfigures(t *testing.T) {
	var bodies = []string{}
	var name1 = "server1.com"
	var name2 = "server2.com"
	var server1 = createServer(name1)
	var server2 = createServer(name2)

	defer server1.Close()
	defer server2.Close()

	var backends = map[string]Backend{
		"something.com": Backend{
			Servers: []Server{},
		},
	}

	lb, err := New(Config{Backends: backends})
	assert.NoError(t, err)

	defer lb.Stop()
	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	go func() {
		serversToChoose := []string{server1.URL, server2.URL}

		for i := 0; i < 100; i++ {
			err := lb.LoadBackends(map[string]Backend{
				"something.com": Backend{
					Servers: []Server{
						{serversToChoose[i%2]},
					},
				},
			})
			assert.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	for i := 0; i < 1000; i++ {
		resp, err := targetHost("something.com", lb.port)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		data, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)
		bodies = append(bodies, string(data))
	}

	assert.Contains(t, bodies, name1)
	assert.Contains(t, bodies, name2)
}

func mustBase64EncodeUser(usr, pwd string) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%s", usr, pwd)))
}

func TestAuthenticate(t *testing.T) {
	var testCases = []struct {
		description string
		users       map[string]string
		headers     map[string]string
		ok          bool
	}{
		{
			description: "authenticates if no correct auth header",
			users:       map[string]string{"admin": "admin"},
			headers: map[string]string{
				"Authorization": "Basic: " + mustBase64EncodeUser("admin", "admin"),
			},
			ok: true,
		},
		{
			description: "doesnt authenticate even if no users set",
			users:       map[string]string{},
			headers:     map[string]string{},
			ok:          false,
		},
		{
			description: "doesnt authenticate if no use of auth header",
			users:       map[string]string{"admin": "admin"},
			headers:     map[string]string{},
			ok:          false,
		},
		{
			description: "doesnt authenticate if invalid auth",
			users:       map[string]string{"admin": "admin"},
			headers: map[string]string{
				"Authorization": "Bearer: token",
			},
			ok: false,
		},
		{
			description: "doesnt authenticate if auth present but user not in set",
			users:       map[string]string{"admin": "admin"},
			headers: map[string]string{
				"Authorization": "Basic: " + mustBase64EncodeUser("myuser", "mypass"),
			},
			ok: false,
		},
		{
			description: "doesnt authenticate if basic not camel cased",
			users:       map[string]string{"admin": "admin"},
			headers: map[string]string{
				"Authorization": "baSIc: " + mustBase64EncodeUser("admin", "admin"),
			},
			ok: false,
		},
		{
			description: "authenticates if auth header not camel cased",
			users:       map[string]string{"admin": "admin"},
			headers: map[string]string{
				"auTHORization": "Basic: " + mustBase64EncodeUser("admin", "admin"),
			},
			ok: true,
		},
	}

	var (
		lb  L7
		req fasthttp.Request
		err error
	)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			lb, err = New(Config{
				Users: tc.users,
			})
			assert.NoError(t, err)

			req = fasthttp.Request{}
			for k, v := range tc.headers {
				req.Header.SetBytesKV(
					[]byte(k),
					[]byte(v))
			}

			assert.Equal(t, tc.ok, lb.authenticate(&fasthttp.RequestCtx{
				Request: req,
			}))
		})
	}
}

func Test_respondsWith401WWAuthenticateIfNotAuthenticated(t *testing.T) {
	var server = createServer("myserver")
	defer server.Close()

	var servers = []Server{
		{
			Address: server.URL,
		},
	}

	lb, err := New(Config{
		Users: map[string]string{
			"admin":"admin",
		},
		Backends: map[string]Backend{
			"something.com": Backend{
				Servers: servers,
			},
		},
	})
	assert.NoError(t, err)

	defer lb.Stop()
	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 10; i++ {
		resp, err := targetHost("something.com", lb.port)
		assert.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("WWW-Authenticate"))
	}
}

func Test_respondsWithAccordinglyIfAuthenticated(t *testing.T) {
	var server = createServer("myserver")
	defer server.Close()

	var servers = []Server{
		{
			Address: server.URL,
		},
	}

	lb, err := New(Config{
		Users: map[string]string{
			"admin":"admin",
		},
		Backends: map[string]Backend{
			"something.com": Backend{
				Servers: servers,
			},
		},
	})
	assert.NoError(t, err)

	defer lb.Stop()
	go func() {
		lb.Listen()
	}()

	time.Sleep(100 * time.Millisecond)


	var addr = fmt.Sprintf("http://localhost:%d", lb.port)
	var client = &http.Client{}

	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return
	}

	req.Header.Add("Authorization","Basic: " + mustBase64EncodeUser("admin", "admin"))
	req.Host = "something.com"

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("WWW-Authenticate"))

	data, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "myserver", string(data))
}
