// Package xep0012 implements XEP-0012 (Last Activity).
//
// It allows to query an entity's idle time/last online time/uptime.
// It also enables answering such requests made to the client.
//
// Using this package necessitates reacting to the synthetic
// LastActivityRequest stanza by replying to it with the correct idle
// time (see (*LastActivityRequest).Reply()).
package last

import (
	"encoding/xml"
	"honnef.co/go/xmpp/client/core"
	"honnef.co/go/xmpp/client/xep/disco"
)

type Conn struct {
	core.Client
}

type LastActivityRequest struct {
	*core.IQ
	c *Conn
}

func init() {
	core.RegisterXEP("last", wrap, "disco")
}

func wrap(c core.Client) (core.XEP, error) {
	conn := &Conn{
		Client: c,
	}

	discovery := conn.MustGetXEP("disco").(*disco.Conn)
	discovery.AddFeature("jabber:iq:last")

	return conn, nil
}

func (c *Conn) Process(stanza core.Stanza) ([]core.Stanza, error) {
	if iq, ok := stanza.(*core.IQ); ok {
		if iq.Query.Space == "jabber:iq:last" && iq.Type == "get" {
			return []core.Stanza{&LastActivityRequest{iq, c}}, nil
		}
	}

	return nil, nil
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
func (c *Conn) Query(who string) (seconds uint64, text string, err error) {
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
