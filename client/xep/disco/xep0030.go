package disco

import (
	"honnef.co/go/xmpp/client/core"

	"encoding/xml"
	"honnef.co/go/xmpp/shared/xep"
	"sync"
)

type Connection struct {
	core.Client
	sync.RWMutex
	stanzas    chan core.Stanza
	identities []Identity
	features   []Feature
}

func init() {
	core.RegisterXEP(30, wrap)
}

func wrap(c core.Client) (xep.Interface, error) {
	conn := &Connection{
		Client:  c,
		stanzas: make(chan core.Stanza, 100),
	}

	conn.AddFeature("http://jabber.org/protocol/disco#info")

	c.SubscribeStanzas(conn.stanzas)
	go conn.read()

	return conn, nil
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
		if iq, ok := stanza.(*core.IQ); ok {
			if iq.Query.Space == "http://jabber.org/protocol/disco#info" && iq.Type == "get" {
				// TODO support queries targetted at nodes
				c.RLock()
				c.SendIQReply(iq, "result", struct {
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

type items struct {
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
func parseInfo(s *core.IQ) (Info, error) {
	var result Info

	if s.IsError() {
		return result, s.Error
	}

	// FIXME handle error
	xml.Unmarshal(s.Inner, &result)

	return result, nil
}

// FIXME return error
func GetInfo(c core.Client, to string) (Info, error) {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
	}{})

	return parseInfo(<-ch)
}

// FIXME return error
func GetInfoFromNode(c core.Client, to, node string) (Info, error) {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
		Node    string   `xml:"node,attr"`
	}{Node: node})

	return parseInfo(<-ch)
}

func (c *Connection) GetItems(to string) ([]Item, error) {
	return GetItems(c, to)
}

func (c *Connection) GetItemsFromNode(to, node string) ([]Item, error) {
	return GetItemsFromNode(c, to, node)
}

func parseItems(s *core.IQ) ([]Item, error) {
	var items items

	if s.IsError() {
		return items.Items, s.Error
	}

	xml.Unmarshal(s.Inner, &items)

	return items.Items, nil
}

func GetItems(c core.Client, to string) ([]Item, error) {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
	}{})

	return parseItems(<-ch)
}

func GetItemsFromNode(c core.Client, to, node string) ([]Item, error) {
	ch, _ := c.SendIQ(to, "get", struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
		Node    string   `xml:"node,attr"`
	}{Node: node})

	return parseItems(<-ch)
}

// TODO do we need the functions or are methods enough?
