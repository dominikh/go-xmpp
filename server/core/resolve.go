package server

import (
	"honnef.co/go/xmpp/shared/core"
)

func Resolve(host string) ([]core.Address, []error) {
	return core.ResolveFQDN(host, "xmpp-server")
}
