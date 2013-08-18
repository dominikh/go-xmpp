package core

import (
	"net"
)

const (
	DefaultClientPort = 5222 // Default port for client-to-server connections
	DefaultServerPort = 5269 // Default port for server-to-server connections
)

// consider renaming this type
type Address struct {
	IPs  []net.IP
	Port int
}

// ResolveFQDN resolves an FQDN to all IP+port pairs to attempt to
// connect to. service must be either xmpp-client or xmpp-server, for
// c2s or s2s connections respectively.
func ResolveFQDN(host, service string) ([]Address, []error) {
	// First attempt using SRV. If that fails for any reason, attempt
	// A/AAAA lookup. All errors will be recorded.
	var errors []error

	_, srvs, err := net.LookupSRV(service, "tcp", host)
	if err != nil {
		ips, err := resolve(host)
		if err != nil {
			return nil, []error{err}
		}

		var port int
		switch service {
		case "xmpp-client":
			port = DefaultClientPort
		case "xmpp-server":
			port = DefaultServerPort
		default:
			panic("invalid service name")
		}

		return []Address{Address{ips, port}}, nil
	}

	if len(srvs) == 1 && srvs[0].Target == "." {
		return nil, nil // FIXME return some sort of error
	}

	addresses := make([]Address, 0, len(srvs))
	for _, srv := range srvs {
		ips, err := resolve(srv.Target)
		if err != nil {
			errors = append(errors, err)
		}
		if len(ips) > 0 {
			addresses = append(addresses, Address{ips, int(srv.Port)})
		}
	}

	return addresses, errors
}

func resolve(host string) ([]net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	return ips, nil
}
