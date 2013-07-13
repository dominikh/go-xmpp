package xep0012

// TODO document that using this requires reacting to
// LastActivityRequest stanzas

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

func Wrap(c rfc6120.Client) *Connection {
	conn := &Connection{
		Client:  c,
		stanzas: make(chan rfc6120.Stanza, 100),
	}
	if !c.RegisterXEP(12, conn, 30) {
		panic("XEP-0012 depends on XEP-0030")
	}

	discovery := conn.MustGetXEP(30).(*xep0030.Connection)
	discovery.AddFeature("jabber:iq:last")

	go conn.read()
	return conn
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

func (t *LastActivityRequest) Reply(seconds uint64) {
	t.c.SendIQReply(t.From, "result", t.ID(), struct {
		XMLName xml.Name `xml:"jabber:iq:last query"`
		Seconds uint64   `xml:"seconds,attr"`
	}{
		Seconds: seconds,
	})
}

func (c *Connection) Query(who string) (seconds uint64, text string) {
	ch, _ := c.SendIQ(who, "get", struct {
		XMLName xml.Name `xml:"jabber:iq:last query"`
	}{})

	res := <-ch
	// TODO handle error case

	var v struct {
		Seconds uint64 `xml:"seconds,attr"`
		Text    string `xml:",chardata"`
	}
	xml.Unmarshal(res.Inner, &v)
	return v.Seconds, v.Text
}
