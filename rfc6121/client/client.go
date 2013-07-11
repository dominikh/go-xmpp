package client

// TODO implement roster versioning

import (
	"encoding/xml"
	"github.com/davecgh/go-spew/spew"
	"honnef.co/go/xmpp/rfc6120/client"
)

var _ = spew.Dump

type Connection struct {
	*client.Connection
	Stream chan client.Stanza
}

func Wrap(c *client.Connection) *Connection {
	conn := &Connection{c, make(chan client.Stanza)}
	go conn.read()
	return conn
}

func (c *Connection) read() {
	for stanza := range c.Connection.Stream {
		spew.Dump(stanza)
		if iq, ok := stanza.(*client.IQ); ok {
			if iq.Query.Space == "jabber:iq:roster" {
				// TODO should we send this stanza back on the stream?
				// After all the user might be interested in it?

				// TODO check 'from' ("Security Warning:
				// Traditionally, a roster push included no 'from'
				// address")
				c.SendIQReply("", "result", stanza.ID(), nil)
			}
		}
		c.Stream <- stanza
	}
}

type Roster []RosterItem

type RosterItem struct {
	JID  string `xml:"jid,attr"`
	Name string `xml:"name,attr,omitempty"`
	// Groups []string // TODO
	Subscription string `xml:"subscription,attr,omitempty"`
}

type rosterQuery struct {
	XMLName xml.Name    `xml:"jabber:iq:roster query"`
	Item    *RosterItem `xml:"item,omitempty"`
}

func (c *Connection) GetRoster() Roster {
	// TODO implement

	ch, _ := c.SendIQ("", "get", rosterQuery{})
	res := <-ch

	spew.Dump(res)

	return nil
}

// AddToRoster adds an item to the roster. If no item with the
// specified JID exists yet, a new one will be created. Otherwise an
// existing one will be updated.
func (c *Connection) AddToRoster(item RosterItem) error {
	ch, _ := c.SendIQ("", "set", rosterQuery{Item: &item})
	// TODO implement error handling
	<-ch
	return nil
}

func (c *Connection) RemoveFromRoster(jid string) error {
	ch, _ := c.SendIQ("", "set", rosterQuery{Item: &RosterItem{
		JID:          jid,
		Subscription: "remove",
	}})
	<-ch
	return nil
	// TODO handle error
}
