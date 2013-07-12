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

type Info struct {
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
func (c Connection) GetInfo(to string) Info {
	return GetInfo(c, to)
}

// FIXME return error
func (c Connection) GetInfoFromNode(to, node string) Info {
	return GetInfoFromNode(c, to, node)
}

// FIXME return error
func parseInfo(s *client.IQ) Info {
	var result Info
	// FIXME handle error
	xml.Unmarshal(s.Inner, &result)

	return result
}

// FIXME return error
func GetInfo(c client.Client, to string) Info {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
	}{})

	return parseInfo(<-ch)
}

// FIXME return error
func GetInfoFromNode(c client.Client, to, node string) Info {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
		Node    string   `xml:"node,attr"`
	}{Node: node})

	return parseInfo(<-ch)
}
