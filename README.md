<h1 align="center">l7 📂  </h1>

<h5 align="center">Minimal HTTP 1.1 load-balancer</h5>

<br/>

[![Build Status](https://travis-ci.org/cirocosta/l7.svg?branch=master)](https://travis-ci.org/cirocosta/l7)


### Overview


`l7` is a load-balancer responsible for performing load-balancing based on parameters passed to it or a configuration file.
It allows you to, in a single line, specify a set of servers and have load sent uniformily to them.


##### Command Line

In order to facilitate testing it's possible to specify all the arguments from the configuration file via parameters to the `l7` command.


```sh
Usage: l7 [--port PORT] [--config CONFIG] [--user USER] [SERVERS [SERVERS ...]]

Positional arguments:
  SERVERS

Options:
  --port PORT, -p PORT   port to listen to [default: 80]
  --config CONFIG, -c CONFIG
                         configuration file to use
  --user USER
  --help, -h             display this help and exit


Example:
  sudo l7 \
        --user admin:admin \		# enforces basic auth on all requests
        --user someone:mypasswd \		
        --port 80 \			# binds to port 80
         mydomain.com=127.0.0.1:8081 \	# list of server configurations
         mydomain.com=127.0.0.1:8082 \
         mydomain.com=127.0.0.1:8083 \
         example.io=127.0.0.1:1337
```

In the example above we make `l7` listen on port `80` and place two rules for load-balancing:
- requests to `mydomain.com` should be split across 3 servers listening on 127.0.0.1
- requests to `example.io` should go to `127.0.0.1:1337`
- every request to either `mydomain.com` or `example.io` must be authenticated


##### Configuration file

The configuration file can be located anywhere on disk. To use one, specify '-c|--config' configuration parameter to the `l7` command:


```sh
l7 --config ./config.yml   # use configuration from ./config.yml
```

The configuration is composed of few definitions. Changing the following example should be enough to get going.

```yaml
# config.yml
port: 80	
users:			# optional
  myuser: 'passwd'
  admin: 'admin'
backends:
  example.com:
    servers:
      - address: 'http://192.168.0.103:8081'
      - address: '//192.168.0.103:8082'         # no 'http(s)://' needed
      - address: 'http://nginx'                 # hostnames can be used
                                                # note.: dns resolution
                                                #        will take place.
```

Above we're specifying that:
- we want `l7` listening on port 80 (this will require using `sudo` - a privileged user - to continue)
- those requests with `host` set to `example.com` should be load-balanced across 3 distinct servers
- all requests must be authenticated with either `myuser:passwd` or `admin:admin`. Note.: this configuration is not required.


Once initialized, the configuration can be reloaded without the need of restarting the whole process. Send a `SIGHUP` to the pid of the load-balancer to reload it on the fly. 

Note.: in the case of errors, `l7` won't crash, but retain the last valid configuration.

To visualize the latest configuration, send a `SIGUSR1` to the process. This will dump to `stdout` the configuration loaded by the `flb`.

