package server

import (
	"honnef.co/go/xmpp/shared/rfc6120"
)

func Resolve(host string) ([]rfc6120.Address, []error) {
	return rfc6120.ResolveFQDN(host, "xmpp-server")
}
