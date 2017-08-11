package lib

import (
	"strings"

	"github.com/pkg/errors"
)

func EqualSeparatedToMap(strs []string) (res map[string][]string, err error) {
	var pair []string
	res = make(map[string][]string)

	for _, str := range strs {
		pair = strings.SplitN(str, "=", 2)
		if len(pair) != 2 {
			err = errors.Errorf(
				"Equal-separated string (%s) "+
					"should produce a pair of string",
				str)
			return
		}

		_, present := res[pair[0]]
		if present {
			res[pair[0]] = append(res[pair[0]], pair[1])
		} else {
			res[pair[0]] = []string{pair[1]}
		}
	}

	return
}

func EqualSeparatedToBackends(list []string) (backends map[string]Backend, err error) {
	if len(list) == 0 {
		err = errors.Errorf(
			"Server list must contain at least 1 server")
		return
	}

	serverMap, err := EqualSeparatedToMap(list)
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't retrieve map from server list %v", list)
		return
	}

	backends = make(map[string]Backend)
	for domain, servers := range serverMap {
		backend := Backend{
			Servers: []Server{},
		}
		for _, address := range servers {
			backend.Servers = append(backend.Servers, Server{
				Address: address,
			})
		}
		backends[domain] = backend
	}

	return
}
