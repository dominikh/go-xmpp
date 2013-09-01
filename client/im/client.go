package im

// TODO implement roster versioning
// TODO handle a roster, that keeps track of presence, the contacts
// who are in it, etc

import (
	"encoding/xml"
	"honnef.co/go/xmpp/client/core"
)

var _ Client = &Conn{}

type Client interface {
	core.Client
	GetRoster() Roster
	AddToRoster(item RosterItem) error
	RemoveFromRoster(jid string) error
	Subscribe(jid string) (cookie string, err error)
	Unsubscribe(jid string) (cookie string, err error)
	ApproveSubscription(auth *AuthorizationRequest)
	DenySubscription(auth *AuthorizationRequest)
	BecomeAvailable()
	BecomeUnavailable()
	SendMessage(typ, to string, message core.Message)
	Reply(orig *core.Message, reply string)
}

func init() {
	core.RegisterXEP("im", wrap)
}

type Conn struct {
	core.Client
}

func wrap(c core.Client) (core.XEP, error) {
	conn := &Conn{
		Client: c,
	}
	return conn, nil
}

func Wrap(c core.Client) *Conn {
	xep, _ := c.RegisterXEP("im")
	return xep.(*Conn)
}

type AuthorizationRequest core.Presence

func (c *Conn) Process(stanza core.Stanza) ([]core.Stanza, error) {
	// TODO way to subscribe to roster events (roster push, subscription requests, ...)
	switch t := stanza.(type) {
	case *core.IQ:
		if t.Query.Space == "jabber:iq:roster" && t.Type == "set" {
			// TODO check 'from' ("Security Warning:
			// Traditionally, a roster push included no 'from'
			// address")
			c.SendIQReply(t, "result", nil)
		}
	case *core.Presence:
		if t.Type == "subscribe" {
			return []core.Stanza{(*AuthorizationRequest)(t)}, nil
		}
	default:
		// TODO track JID etc
	}

	return nil, nil
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

func (c *Conn) GetRoster() Roster {
	// TODO implement

	ch, _ := c.SendIQ("", "get", rosterQuery{})
	<-ch

	return nil
}

// AddToRoster adds an item to the roster. If no item with the
// specified JID exists yet, a new one will be created. Otherwise an
// existing one will be updated.
func (c *Conn) AddToRoster(item RosterItem) error {
	ch, _ := c.SendIQ("", "set", rosterQuery{Item: &item})
	// TODO implement error handling
	<-ch
	return nil
}

func (c *Conn) RemoveFromRoster(jid string) error {
	ch, _ := c.SendIQ("", "set", rosterQuery{Item: &RosterItem{
		JID:          jid,
		Subscription: "remove",
	}})
	<-ch
	return nil
	// TODO handle error
}

func (c *Conn) Subscribe(jid string) (cookie string, err error) {
	cookie, err = c.SendPresence(core.Presence{
		Header: core.Header{
			To:   jid,
			Type: "subscribe",
		},
	})
	return
	// TODO handle error
}

func (c *Conn) Unsubscribe(jid string) (cookie string, err error) {
	cookie, err = c.SendPresence(core.Presence{
		Header: core.Header{
			To:   jid,
			Type: "unsubscribe",
		},
	})
	return
	// TODO handle error
}

func (c *Conn) ApproveSubscription(auth *AuthorizationRequest) {
	c.SendPresence(core.Presence{
		Header: core.Header{
			To:   auth.From,
			Type: "subscribed",
		},
	})
}

func (c *Conn) PreapproveSubscription(jid string) error {
	if !c.Features().Includes("sub") {
		// FIXME return error
	}

	c.SendPresence(core.Presence{
		Header: core.Header{
			To:   jid,
			Type: "subscribed",
		},
	})

	return nil
}

func (c *Conn) DenySubscription(auth *AuthorizationRequest) {
	c.SendPresence(core.Presence{
		Header: core.Header{
			To:   auth.From,
			Type: "unsubscribed",
		},
	})
}

func (c *Conn) RevokeSubscription(jid string) {
	c.SendPresence(core.Presence{
		Header: core.Header{
			To:   jid,
			Type: "unsubscribed",
		},
	})
}

func (c *Conn) BecomeAvailable() {
	// TODO document SendPresence (rfc6120) for more specific needs
	c.SendPresence(core.Presence{})
}

func (c *Conn) BecomeUnavailable() {
	// TODO document SendPresence (rfc6120) for more specific needs
	c.Encode(core.Presence{Header: core.Header{Type: "unavailable"}})
}

func (c *Conn) SendMessage(typ, to string, message core.Message) {
	// TODO support extended items in the mssage
	// TODO if `to` is a bare JID, see if we know about a full JID to
	// use instead. if it's a full jid, check if it's outdated.
	// Probably make these two things explicit by providing a function
	// on the roster that the user has to call, that translates a jid
	// into a better one. replying should probably automatically use
	// it.
	message.Header = core.Header{
		From: c.JID(),
		To:   to,
		Type: typ,
	}

	c.Encode(message)
}

func (c *Conn) Reply(orig *core.Message, reply string) {
	// TODO use bare JID if full JID isn't up to date anymore
	// TODO support subject
	// TODO support extended items
	c.SendMessage(orig.Type, orig.From, core.Message{Body: reply, Thread: orig.Thread})
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
