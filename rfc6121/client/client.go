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

type Client interface {
	client.Client
	GetRoster() Roster
	AddToRoster(item RosterItem) error
	RemoveFromRoster(jid string) error
	Subscribe(jid string) (cookie string, err error)
	Unsubscribe(jid string) (cookie string, err error)
	ApproveSubscription(jid string)
	DenySubscription(jid string)
	BecomeAvailable()
	BecomeUnavailable()
	SendMessage(typ, to, message string)
	Reply(orig *client.Message, reply string)
}

type connection struct {
	client.Client
	stanzas chan client.Stanza
}

func Wrap(c client.Client) Client {
	conn := &connection{
		Client:  c,
		stanzas: make(chan client.Stanza, 100),
	}
	go conn.read()
	c.SubscribeStanzas(conn.stanzas)
	return conn
}

type AuthorizationRequest client.Presence

func (c *connection) read() {
	for stanza := range c.stanzas {
		// TODO way to subscribe to roster events (roster push, subscription requests, ...)
		switch t := stanza.(type) {
		case *client.IQ:
			if t.Query.Space == "jabber:iq:roster" && t.Type == "set" {
				// TODO check 'from' ("Security Warning:
				// Traditionally, a roster push included no 'from'
				// address")
				c.SendIQReply("", "result", stanza.ID(), nil)
			}
		case *client.Presence:
			if t.Type == "subscribe" {
				c.EmitStanza((*AuthorizationRequest)(t))
				// c.subscribers.send((*AuthorizationRequest)(t))
			}
		default:
			// TODO track JID etc
		}
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

func (c *connection) GetRoster() Roster {
	// TODO implement

	ch, _ := c.SendIQ("", "get", rosterQuery{})
	<-ch

	return nil
}

// AddToRoster adds an item to the roster. If no item with the
// specified JID exists yet, a new one will be created. Otherwise an
// existing one will be updated.
func (c *connection) AddToRoster(item RosterItem) error {
	ch, _ := c.SendIQ("", "set", rosterQuery{Item: &item})
	// TODO implement error handling
	<-ch
	return nil
}

func (c *connection) RemoveFromRoster(jid string) error {
	ch, _ := c.SendIQ("", "set", rosterQuery{Item: &RosterItem{
		JID:          jid,
		Subscription: "remove",
	}})
	<-ch
	return nil
	// TODO handle error
}

func (c *connection) Subscribe(jid string) (cookie string, err error) {
	cookie, err = c.SendPresence(client.Presence{
		Header: client.Header{
			To:   jid,
			Type: "subscribe",
		},
	})
	return
	// TODO handle error
}

func (c *connection) Unsubscribe(jid string) (cookie string, err error) {
	cookie, err = c.SendPresence(client.Presence{
		Header: client.Header{
			To:   jid,
			Type: "unsubscribe",
		},
	})
	return
	// TODO handle error
}

func (c *connection) ApproveSubscription(jid string) {
	c.SendPresence(client.Presence{
		Header: client.Header{
			To:   jid,
			Type: "subscribed",
		},
	})
}

func (c *connection) DenySubscription(jid string) {
	// TODO document that this can also be used to revoke an existing
	// subscription
	c.SendPresence(client.Presence{
		Header: client.Header{
			To:   jid,
			Type: "unsubscribed",
		},
	})
}

func (c *connection) BecomeAvailable() {
	// TODO document SendPresence (rfc6120) for more specific needs
	c.SendPresence(client.Presence{})
}

func (c *connection) BecomeUnavailable() {
	// TODO document SendPresence (rfc6120) for more specific needs
	// TODO can't be have one global xml encoder?
	xml.NewEncoder(c).Encode(client.Presence{Header: client.Header{Type: "unavailable"}})
}

func (c *connection) SendMessage(typ, to, message string) {
	// TODO support extended items in the mssage
	// TODO if `to` is a bare JID, see if we know about a full JID to
	// use instead
	// TODO actually keep track of JIDs
	// TODO support <thread>
	// TODO support subject

	m := client.Message{
		Header: client.Header{
			From: c.JID(),
			To:   to,
			Type: typ,
		},
		Body: message,
	}

	xml.NewEncoder(c).Encode(m)
}

func (c *connection) Reply(orig *client.Message, reply string) {
	// TODO threading
	// TODO use bare JID if full JID isn't up to date anymore
	// TODO support subject
	// TODO support extended items
	c.SendMessage(orig.Type, orig.From, reply)
}

// The user's client SHOULD address the initial message in a chat
// session to the bare JID <contact@domainpart> of the contact (rather
// than attempting to guess an appropriate full JID
// <contact@domainpart/resourcepart> based on the <show/>, <status/>,
// or <priority/> value of any presence notifications it might have
// received from the contact). Until and unless the user's client
// receives a reply from the contact, it SHOULD send any further
// messages to the contact's bare JID. The contact's client SHOULD
// address its replies to the user's full JID
// <user@domainpart/resourcepart> as provided in the 'from' address of
// the initial message. Once the user's client receives a reply from
// the contact's full JID, it SHOULD address its subsequent messages
// to the contact's full JID as provided in the 'from' address of the
// contact's replies, thus "locking in" on that full JID. A client
// SHOULD "unlock" after having received a <message/> or <presence/>
// stanza from any other resource controlled by the peer (or a
// presence stanza from the locked resource); as a result, it SHOULD
// address its next message(s) in the chat session to the bare JID of
// the peer (thus "unlocking" the previous "lock") until it receives a
// message from one of the peer's full JIDs.

// When two parties engage in a chat session but do not share presence
// with each other based on a presence subscription, they SHOULD send
// directed presence to each other so that either party can easily
// discover if the peer goes offline during the course of the chat
// session. However, a client MUST provide a way for a user to disable
// such presence sharing globally or to enable it only with particular
// entities. Furthermore, a party SHOULD send directed unavailable
// presence to the peer when it has reason to believe that the chat
// session is over (e.g., if, after some reasonable amount of time, no
// subsequent messages have been exchanged between the parties).
