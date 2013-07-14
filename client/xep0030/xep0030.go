package xep0030

import (
	"honnef.co/go/xmpp/client/rfc6120"

	"encoding/xml"
	"sync"
)

type Connection struct {
	rfc6120.Client
	sync.RWMutex
	stanzas    chan rfc6120.Stanza
	identities []Identity
	features   []Feature
}

// TODO reconsider the `Wrap` name
func Wrap(c rfc6120.Client) *Connection {
	conn := &Connection{
		Client:  c,
		stanzas: make(chan rfc6120.Stanza, 100),
	}

	conn.AddFeature("http://jabber.org/protocol/disco#info")

	c.SubscribeStanzas(conn.stanzas)

	go conn.read()

	c.RegisterXEP(30, conn)

	return conn
}

func (c *Connection) AddIdentity(id Identity) {
	c.Lock()
	c.identities = append(c.identities, id)
	c.Unlock()
}

func (c *Connection) AddFeature(f string) {
	c.Lock()
	c.features = append(c.features, Feature{f})
	c.Unlock()
}

func (c *Connection) read() {
	// TODO support queries for items/item nodes
	for stanza := range c.stanzas {
		if iq, ok := stanza.(*rfc6120.IQ); ok {
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
func (c *Connection) GetInfo(to string) (Info, error) {
	return GetInfo(c, to)
}

// FIXME return error
func (c *Connection) GetInfoFromNode(to, node string) (Info, error) {
	return GetInfoFromNode(c, to, node)
}

// FIXME return error
func parseInfo(s *rfc6120.IQ) (Info, error) {
	var result Info

	if s.IsError() {
		return result, s.Error
	}

	// FIXME handle error
	xml.Unmarshal(s.Inner, &result)

	return result, nil
}

// FIXME return error
func GetInfo(c rfc6120.Client, to string) (Info, error) {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
	}{})

	return parseInfo(<-ch)
}

// FIXME return error
func GetInfoFromNode(c rfc6120.Client, to, node string) (Info, error) {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
		Node    string   `xml:"node,attr"`
	}{Node: node})

	return parseInfo(<-ch)
}

func (c *Connection) GetItems(to string) (Items, error) {
	return GetItems(c, to)
}

func (c *Connection) GetItemsFromNode(to, node string) (Items, error) {
	return GetItemsFromNode(c, to, node)
}

func parseItems(s *rfc6120.IQ) (Items, error) {
	var items Items

	if s.IsError() {
		return items, s.Error
	}

	xml.Unmarshal(s.Inner, &items)

	return items, nil
}

func GetItems(c rfc6120.Client, to string) (Items, error) {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
	}{})

	return parseItems(<-ch)
}

func GetItemsFromNode(c rfc6120.Client, to, node string) (Items, error) {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
		Node    string   `xml:"node,attr"`
	}{Node: node})

	return parseItems(<-ch)
}

// TODO do we need the functions or are methods enough?
