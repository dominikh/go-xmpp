package rfc6120 // TODO probably rename this

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

// ResolveC2S resolves an FQDN to all IP+port pairs to attempt to
// connect to for client-to-server connections.
func ResolveC2S(host string) ([]Address, []error) {
	return resolveFQDN(host, "xmpp-client")
}

// ResolveS2S resolves an FQDN to all IP+port pairs to attempt to
// connect to for server-to-server connections.
func ResolveS2S(host string) ([]Address, []error) {
	return resolveFQDN(host, "xmpp-server")
}

func resolveFQDN(host, service string) ([]Address, []error) {
	// First attempt using SRV. If that fails for any reason, attempt
	// A/AAAA lookup. All errors will be recorded.
	var errors []error

	_, srvs, err := net.LookupSRV(service, "tcp", host)
	if err != nil {
		errors = append(errors, err)

		ips, err := resolve(host)
		if err != nil {
			errors = append(errors, err)
			return nil, errors
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

		return []Address{Address{ips, port}}, errors
	}

	if len(srvs) == 1 && srvs[0].Target == "." {
		return nil, nil
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

	return addresses, nil
}

func resolve(host string) ([]net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	return ips, nil
}
