package xep0012

import (
	"encoding/xml"
	"honnef.co/go/xmpp/client/rfc6120"
	"honnef.co/go/xmpp/client/xep0030"
	"sync/atomic"
)

type Connection struct {
	seconds uint64
	rfc6120.Client
	stanzas chan rfc6120.Stanza
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

// TODO would a callback be better here? Imagine someone asks for this
// client's idle time, how are we going to get an accurate value?
func (c *Connection) SetLast(seconds uint64) {
	atomic.StoreUint64(&c.seconds, seconds)
}

func (c *Connection) read() {
	for stanza := range c.stanzas {
		if iq, ok := stanza.(*rfc6120.IQ); ok {
			if iq.Query.Space == "jabber:iq:last" && iq.Type == "get" {
				c.SendIQReply(iq.From, "result", iq.ID(), struct {
					XMLName xml.Name `xml:"jabber:iq:last query"`
					Seconds uint64   `xml:"seconds,attr"`
				}{
					Seconds: atomic.LoadUint64(&c.seconds),
				})
			}
		}
	}
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
