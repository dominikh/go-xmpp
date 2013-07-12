package xep0030

import (
	"encoding/xml"
	"honnef.co/go/xmpp/rfc6120/client"
)

type Connection struct {
	client.Client
}

// TODO reconsider the `Wrap` name
func Wrap(c client.Client) Connection {
	return Connection{c}
}

type Result struct {
	Identities []Identity `xml:"identity"`
	Features   []Feature  `xml:"feature"`
}

type Identity struct {
	Category string `xml:"category,attr"`
	Type     string `xml:"type,attr"`
	Name     string `xml:"name,attr"`
}

type Feature struct {
	Var string `xml:"var,attr"`
}

// FIXME return error
func (c Connection) Query(to string) Result {
	return Query(c, to)
}

// FIXME return error
func (c Connection) QueryNode(to, node string) Result {
	return QueryNode(c, to, node)
}

// FIXME return error
func parseQueryResult(s *client.IQ) Result {
	var result Result
	// FIXME handle error
	xml.Unmarshal(s.Inner, &result)

	return result
}

// FIXME return error
func Query(c client.Client, to string) Result {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
	}{})

	return parseQueryResult(<-ch)
}

// FIXME return error
func QueryNode(c client.Client, to, node string) Result {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
		Node    string   `xml:"node,attr"`
	}{Node: node})

	return parseQueryResult(<-ch)
}
