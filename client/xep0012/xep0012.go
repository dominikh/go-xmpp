// Package xep0012 implements XEP-0012 (Last Activity).
//
// It allows to query an entities idle time/last online time/uptime.
// It also enables answering such requests made to the client.
//
// Using this package necessitates reacting to the synthetic
// LastActivityRequest stanza by replying to it with the correct idle
// time (see (*LastActivityRequest).Reply()).
package xep0012

import (
	"encoding/xml"
	"honnef.co/go/xmpp/client/rfc6120"
	"honnef.co/go/xmpp/client/xep0030"
)

type Connection struct {
	rfc6120.Client
	stanzas chan rfc6120.Stanza
}

type LastActivityRequest struct {
	*rfc6120.IQ
	c *Connection
}

func init() {
	rfc6120.RegisterXEP(12, Wrap)
}

func Wrap(c rfc6120.Client) error {
	conn := &Connection{
		Client:  c,
		stanzas: make(chan rfc6120.Stanza, 100),
	}

	if err := c.RegisterXEP(12, conn, 30); err != nil {
		return err
	}

	discovery := conn.MustGetXEP(30).(*xep0030.Connection)
	discovery.AddFeature("jabber:iq:last")

	c.SubscribeStanzas(conn.stanzas)
	go conn.read()

	return nil
}

func (c *Connection) read() {
	for stanza := range c.stanzas {
		if iq, ok := stanza.(*rfc6120.IQ); ok {
			if iq.Query.Space == "jabber:iq:last" && iq.Type == "get" {
				c.EmitStanza(&LastActivityRequest{iq, c})
			}
		}
	}
}

// Reply replies to the Last Activity query.
func Reply(t *LastActivityRequest, seconds uint64) {
	t.c.SendIQReply(t.IQ, "result", struct {
		XMLName xml.Name `xml:"jabber:iq:last query"`
		Seconds uint64   `xml:"seconds,attr"`
	}{
		Seconds: seconds,
	})
}

// Query sends a Last Activity query to an entity. The interpretation
// of the returned values depends on whether the entity is an account,
// resource or service.
func (c *Connection) Query(who string) (seconds uint64, text string, err error) {
	ch, _ := c.SendIQ(who, "get", struct {
		XMLName xml.Name `xml:"jabber:iq:last query"`
	}{})

	res := <-ch
	if res.IsError() {
		return 0, "", res.Error
	}

	var v struct {
		Seconds uint64 `xml:"seconds,attr"`
		Text    string `xml:",chardata"`
	}

	// TODO consider wrapping this error in a more descriptive type
	err = xml.Unmarshal(res.Inner, &v)
	return v.Seconds, v.Text, err
}
