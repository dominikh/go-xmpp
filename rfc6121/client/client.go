package client

// TODO implement roster versioning
// TODO implement pre-approval
// TODO handle a roster, that keeps track of presence, the contacts
// who are in it, etc

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
		// TODO maybe have a channel for Roster events (roster push)
		// on which to send changes?

		// TODO maybe have a channel for subscription requests?
		if iq, ok := stanza.(*client.IQ); ok {
			if iq.Query.Space == "jabber:iq:roster" && iq.Type == "set" {
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
	<-ch

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

func (c *Connection) Subscribe(jid string) (cookie string, err error) {
	cookie, err = c.Connection.SendPresence(client.Presence{
		Header: client.Header{
			To:   jid,
			Type: "subscribe",
		},
	})
	return
	// TODO handle error
}

func (c *Connection) Unsubscribe(jid string) (cookie string, err error) {
	cookie, err = c.Connection.SendPresence(client.Presence{
		Header: client.Header{
			To:   jid,
			Type: "unsubscribe",
		},
	})
	return
	// TODO handle error
}

func (c *Connection) ApproveSubscription(jid string) {
	c.Connection.SendPresence(client.Presence{
		Header: client.Header{
			To:   jid,
			Type: "subscribed",
		},
	})
}

func (c *Connection) DenySubscription(jid string) {
	// TODO document that this can also be used to revoke an existing
	// subscription
	c.Connection.SendPresence(client.Presence{
		Header: client.Header{
			To:   jid,
			Type: "unsubscribed",
		},
	})
}

func (c *Connection) SendPresence(p *client.Presence) {
	var pp client.Presence
	if p != nil {
		pp = *p
	}

	xml.NewEncoder(c).Encode(pp)
}

func (c *Connection) BecomeUnavailable(p *client.Presence) {
	var pp client.Presence
	if p != nil {
		pp = *p
	}

	pp.Type = "unavailable"
	pp.Show = ""
	pp.Priority = 0
	// TODO can't be have one global xml encoder?
	xml.NewEncoder(c).Encode(pp)
}
