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

// TODO consider not swallowing the error. It might be caused by e.g.
// network issues, simply not returning any addresses would be
// confusing to the user.

// ResolveC2S resolves an FQDN to all IP+port pairs to attempt to
// connect to for client-to-server connections.
func ResolveC2S(host string) []Address {
	return resolveFQDN(host, "xmpp-client")
}

// ResolveS2S resolves an FQDN to all IP+port pairs to attempt to
// connect to for server-to-server connections.
func ResolveS2S(host string) []Address {
	return resolveFQDN(host, "xmpp-server")
}

func resolveFQDN(host, service string) []Address {
	_, srvs, err := net.LookupSRV(service, "tcp", host)
	if err != nil {
		// TODO use fallback
		ips := resolve(host)
		var port int
		switch service {
		case "xmpp-client":
			port = DefaultClientPort
		case "xmpp-server":
			port = DefaultServerPort
		default:
			panic("invalid service name")
		}

		return []Address{Address{ips, port}}
	}

	if len(srvs) == 1 && srvs[0].Target == "." {
		return nil
	}

	addresses := make([]Address, 0, len(srvs))
	for _, srv := range srvs {
		ips := resolve(srv.Target)
		if len(ips) > 0 {
			addresses = append(addresses, Address{ips, int(srv.Port)})
		}
	}

	return addresses
}

func resolve(host string) []net.IP {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil
	}

	return ips
}
