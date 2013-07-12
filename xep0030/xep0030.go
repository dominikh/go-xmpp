package xep0030

import (
	"encoding/xml"
	"honnef.co/go/xmpp/rfc6120/client"
	"sync"
)

type Connection struct {
	client.Client
	sync.RWMutex
	stanzas    chan client.Stanza
	identities []Identity
	features   []Feature
}

// TODO reconsider the `Wrap` name
func Wrap(c client.Client) *Connection {
	conn := &Connection{
		Client:  c,
		stanzas: make(chan client.Stanza, 100),
	}

	conn.AddFeature(Feature{Var: "http://jabber.org/protocol/disco#info"})

	c.SubscribeStanzas(conn.stanzas)

	go conn.read()
	return conn
}

func (c *Connection) AddIdentity(id Identity) {
	c.Lock()
	c.identities = append(c.identities, id)
	c.Unlock()
}

func (c *Connection) AddFeature(f Feature) {
	c.Lock()
	c.features = append(c.features, f)
	c.Unlock()
}

func (c *Connection) read() {
	// TODO support queries for items/item nodes
	for stanza := range c.stanzas {
		if iq, ok := stanza.(*client.IQ); ok {
			if iq.Query.Space == "http://jabber.org/protocol/disco#info" && iq.Type == "get" {
				// TODO support queries targetted at nodes
				c.RLock()
				c.SendIQReply(iq.From, "result", iq.ID(), struct {
					XMLName    xml.Name   `xml:"http://jabber.org/protocol/disco#info query"`
					Identities []Identity `xml:"identity"`
					Features   []Feature  `xml:"feature"`
				}{
					Identities: c.identities,
					Features:   c.features,
				})
				c.RUnlock()
			}
		}
	}
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

type Items struct {
	Items []Item `xml:"item"`
}

type Item struct {
	JID  string `xml:"jid,attr"`
	Name string `xml:"name,attr"`
	Node string `xml:"node,attr"`
}

// FIXME return error
func (c *Connection) GetInfo(to string) Info {
	return GetInfo(c, to)
}

// FIXME return error
func (c *Connection) GetInfoFromNode(to, node string) Info {
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

func (c *Connection) GetItems(to string) Items {
	return GetItems(c, to)
}

func (c *Connection) GetItemsFromNode(to, node string) Items {
	return GetItemsFromNode(c, to, node)
}

func parseItems(s *client.IQ) Items {
	var items Items
	xml.Unmarshal(s.Inner, &items)

	return items
}

func GetItems(c client.Client, to string) Items {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
	}{})

	return parseItems(<-ch)
}

func GetItemsFromNode(c client.Client, to, node string) Items {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
		Node    string   `xml:"node,attr"`
	}{Node: node})

	return parseItems(<-ch)
}
