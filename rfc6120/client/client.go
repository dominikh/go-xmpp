package client

// TODO make sure whitespace keepalive doesn't break our code
// TODO check namespaces everywhere
// TODO optional reconnect handling: 1) reconnect if enabled 2) close
// channels when the connection is gone for good

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"net"
	"strings"
	"sync"
)

const (
	nsStream  = "http://etherx.jabber.org/streams"
	nsTLS     = "urn:ietf:params:xml:ns:xmpp-tls"
	nsSASL    = "urn:ietf:params:xml:ns:xmpp-sasl"
	nsBind    = "urn:ietf:params:xml:ns:xmpp-bind"
	nsSession = "urn:ietf:params:xml:ns:xmpp-session"
	nsClient  = "jabber:client"
)

var _ = spew.Dump

var SupportedMechanisms = []string{"PLAIN"}

// TODO move out of client package?
func findCompatibleMechanism(ours, theirs []string) string {
	for _, our := range ours {
		for _, their := range theirs {
			if our == their {
				return our
			}
		}
	}

	return ""
}

type subscribers struct {
	sync.RWMutex
	chans []chan<- Stanza
}

func (s *subscribers) send(stanza Stanza) {
	s.RLock()
	defer s.RUnlock()
	for _, ch := range s.chans {
		select {
		case ch <- stanza:
		default:
		}
	}
}

func (s *subscribers) subscribe(ch chan<- Stanza) {
	s.Lock()
	defer s.Unlock()
	s.chans = append(s.chans, ch)
}

type Connection struct {
	net.Conn
	sync.Mutex
	User       string
	Host       string
	decoder    *xml.Decoder
	Features   Features
	Password   string
	cookie     <-chan string
	cookieQuit chan<- struct{}
	JID        string
	callbacks  map[string]chan *IQ
	Closing    bool
	// TODO reconsider choice of structure when we allow unsubscribing
	subscribers subscribers
}

func generateCookies(ch chan<- string, quit <-chan struct{}) {
	id := uint64(0)
	for {
		select {
		case ch <- fmt.Sprintf("%d", id):
			id++
		case <-quit:
			return
		}
	}
}

func Dial(user, host, password string) (conn *Connection, errors []error, ok bool) {
	addrs, errors := Resolve(host)

connectLoop:
	for _, addr := range addrs {
		for _, ip := range addr.IPs {
			c, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: ip, Port: addr.Port})
			if err != nil {
				errors = append(errors, err)
				continue
			} else {
				cookieChan := make(chan string)
				cookieQuitChan := make(chan struct{})
				go generateCookies(cookieChan, cookieQuitChan)
				conn = &Connection{
					Conn:       c,
					User:       user,
					Password:   password,
					Host:       host,
					decoder:    xml.NewDecoder(c),
					cookie:     cookieChan,
					cookieQuit: cookieQuitChan,
					callbacks:  make(map[string]chan *IQ),
				}

				break connectLoop
			}
		}
	}

	if conn == nil {
		return nil, errors, false
	}

	// TODO error handling
	for {
		conn.OpenStream()
		conn.ReceiveStream()
		conn.ParseFeatures()
		if conn.Features.Includes("starttls") {
			conn.StartTLS() // TODO handle error
			continue
		}

		if conn.Features.Requires("sasl") {
			conn.SASL()
			continue
		}
		break
	}

	go conn.read()
	conn.Bind()

	return conn, errors, true
}

type Stanza interface {
	ID() string
	IsError() bool
}

type Header struct {
	From string `xml:"from,attr,omitempty"`
	Id   string `xml:"id,attr,omitempty"`
	To   string `xml:"to,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

func (h Header) ID() string {
	return h.Id
}

func (Header) IsError() bool {
	return false
}

type Message struct {
	XMLName xml.Name `xml:"jabber:client message"`
	Header

	Subject string `xml:"subject"`
	Body    string `xml:"body"`
	Thread  string `xml:"thread"`
}

type Text struct {
	Lang string `xml:"lang,attr"`
	Body string `xml:",chardata"`
}

type Presence struct {
	XMLName xml.Name `xml:"jabber:client presence"`
	Header

	Lang string `xml:"lang,attr,omitempty"`

	Show     string `xml:"show,omitempty"`
	Status   string `xml:"status,omitempty"`
	Priority int    `xml:"priority,omitempty"`
	Error    *Error `xml:"error,omitempty"`
	// TODO support other tags inside the presence
}

type IQ struct { // info/query
	XMLName xml.Name `xml:"jabber:client iq"`
	Header

	Error *Error   `xml:"error"`
	Query xml.Name `xml:"query"`
	Inner []byte   `xml:",innerxml"`
}

type Error struct {
	XMLName xml.Name `xml:"jabber:client error"`
	Code    string   `xml:"code,attr"`
	Type    string   `xml:"type,attr"`
	Any     xml.Name `xml:",any"`
	Text    string   `xml:"text"`
}

type streamError struct {
	XMLName xml.Name `xml:"http://etherx.jabber.org/streams error"`
	Any     xml.Name `xml:",any"`
	Text    string   `xml:"text"`
}

func (Error) ID() string {
	return ""
}

func (Error) IsError() bool {
	return true
}

func (streamError) ID() string {
	return ""
}

func (streamError) IsError() bool {
	return true
}

func (c *Connection) read() {
	for {
		t, _ := c.NextStartElement()

		if t == nil {
			c.Lock()
			for _, ch := range c.callbacks {
				close(ch)
			}
			c.Close()
			return
		}

		var nv Stanza
		switch t.Name.Space + " " + t.Name.Local {
		case nsStream + " error":
			nv = &streamError{}
		case nsClient + " message":
			nv = &Message{}
		case nsClient + " presence":
			nv = &Presence{}
		case nsClient + " iq":
			nv = &IQ{}
		case nsClient + " error":
			nv = &Error{}
		default:
			fmt.Println(t.Name.Local)
			// TODO handle error
		}

		// Unmarshal into that storage.
		c.decoder.DecodeElement(nv, t)
		if iq, ok := nv.(*IQ); ok && (iq.Type == "result" || iq.Type == "error") {
			c.Lock()
			if ch, ok := c.callbacks[nv.ID()]; ok {
				ch <- iq
				delete(c.callbacks, nv.ID())
			}
			c.Unlock()
		} else {
			c.subscribers.send(nv)
		}
	}
}

func (c *Connection) getCookie() string {
	return <-c.cookie
}

func (c *Connection) Bind() {
	// TODO support binding to a user-specified resource
	// TODO handle error cases

	ch, _ := c.SendIQ("", "set", struct {
		XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	}{})
	response := <-ch
	var bind struct {
		XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
		Resource string   `xml:"resource"`
		JID      string   `xml:"jid"`
	}

	xml.Unmarshal(response.Inner, &bind)
	c.JID = bind.JID
}

func (c *Connection) Reset() {
	c.decoder = xml.NewDecoder(c.Conn)
	c.Features = nil
}

func (c *Connection) SASL() {
	payload := fmt.Sprintf("\x00%s\x00%s", c.User, c.Password)
	payloadb64 := base64.StdEncoding.EncodeToString([]byte(payload))
	fmt.Fprintf(c, "<auth xmlns='urn:ietf:params:xml:ns:xmpp-sasl' mechanism='PLAIN'>%s</auth>", payloadb64)
	t, _ := c.NextStartElement() // FIXME error handling
	if t.Name.Local == "success" {
		c.Reset()
	} else {
		// TODO handle the error case
	}

	// TODO actually determine which mechanism we can use, use interfaces etc to call it
}

func (c *Connection) StartTLS() error {
	fmt.Fprint(c, "<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>")
	t, _ := c.NextStartElement() // FIXME error handling
	if t.Name.Local != "proceed" {
		// TODO handle this. this should be <failure>, and the server
		// will close the connection on us.
	}

	tlsConn := tls.Client(c.Conn, nil)
	if err := tlsConn.Handshake(); err != nil {
		return err
	}

	tlsState := tlsConn.ConnectionState()
	if len(tlsState.VerifiedChains) == 0 {
		return errors.New("xmpp: failed to verify TLS certificate") // FIXME
	}

	if err := tlsConn.VerifyHostname(c.Host); err != nil {
		return errors.New("xmpp: failed to match TLS certificate to name: " + err.Error()) // FIXME
	}

	c.Conn = tlsConn
	c.Reset()

	return nil
}

// TODO Move this outside of client. This function will be used by
// servers, too.
func (c *Connection) NextStartElement() (*xml.StartElement, error) {
	for {
		t, err := c.decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := t.(type) {
		case xml.StartElement:
			return &t, nil
		case xml.EndElement:
			if t.Name.Local == "stream" && t.Name.Space == nsStream {
				return nil, nil
			}
		}
	}
}

func (c *Connection) NextToken() (xml.Token, error) {
	return c.decoder.Token()
}

type UnexpectedMessage struct {
	Name string
}

func (e UnexpectedMessage) Error() string {
	return e.Name
}

// TODO return error of Fprintf
func (c *Connection) OpenStream() {
	// TODO consider not including the JID if the connection isn't encrypted yet
	// TODO configurable xml:lang
	fmt.Fprintf(c, "<?xml version='1.0' encoding='UTF-8'?><stream:stream from='%s@%s' to='%s' version='1.0' xml:lang='en' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>",
		c.User, c.Host, c.Host)
}

type UnsupportedVersion struct {
	Version string
}

func (e UnsupportedVersion) Error() string {
	return "Unsupported XMPP version: " + e.Version
}

func (c *Connection) ReceiveStream() error {
	t, err := c.NextStartElement() // TODO error handling
	if err != nil {
		return err
	}

	if t.Name.Local != "stream" {
		return UnexpectedMessage{t.Name.Local}
	}

	if t.Name.Space != "http://etherx.jabber.org/streams" {
		// TODO consider a function for sending errors
		fmt.Fprint(c, "<stream:error><invalid-namespace xmlns='urn:ietf:params:xml:ns:xmpp-streams'/>")
		c.Close()
		// FIXME return error
		return nil // FIXME do we need to skip over any tokens here?
	}

	var version string
	for _, attr := range t.Attr {
		switch attr.Name.Local {
		// TODO consider storing all attributes in a Stream struct
		case "version":
			version = attr.Value
		}
	}

	if version == "" {
		return UnsupportedVersion{"0.9"}
	}

	parts := strings.Split(version, ".")
	if parts[0] != "1" {
		return UnsupportedVersion{version}
	}

	return nil
}

func (c *Connection) Close() {
	if c.Closing {
		// Terminate TCP connection
		c.Conn.Close()
		return
	}

	fmt.Fprint(c, "</stream:stream>")
	c.Closing = true
	// TODO implement timeout for waiting on </stream> from other end

	// TODO "to help prevent a truncation attack the party that is
	// closing the stream MUST send a TLS close_notify alert and MUST
	// receive a responding close_notify alert from the other party
	// before terminating the underlying TCP connection"
}

var xmlSpecial = map[byte]string{
	'<':  "&lt;",
	'>':  "&gt;",
	'"':  "&quot;",
	'\'': "&apos;",
	'&':  "&amp;",
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	for i := 0; i < len(s); i++ {
		c := s[i]
		if s, ok := xmlSpecial[c]; ok {
			b.WriteString(s)
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// TODO error handling
func (c *Connection) SendIQ(to, typ string, value interface{}) (chan *IQ, string) {
	cookie := c.getCookie()
	reply := make(chan *IQ, 1)
	c.Lock()
	c.callbacks[cookie] = reply
	c.Unlock()

	toAttr := ""
	if len(to) > 0 {
		toAttr = "to='" + xmlEscape(to) + "'"
	}

	fmt.Fprintf(c, "<iq %s from='%s' type='%s' id='%s'>", toAttr, xmlEscape(c.JID), xmlEscape(typ), cookie)
	xml.NewEncoder(c).Encode(value)
	fmt.Fprintf(c, "</iq>")

	return reply, cookie
}

func (c *Connection) SendIQReply(to, typ, id string, value interface{}) {
	toAttr := ""
	if len(to) > 0 {
		toAttr = "to='" + xmlEscape(to) + "'"
	}

	fmt.Fprintf(c, "<iq %s from='%s' type='%s' id='%s'>", toAttr, xmlEscape(c.JID), xmlEscape(typ), id)
	if value != nil {
		xml.NewEncoder(c).Encode(value)
	}
	fmt.Fprintf(c, "</iq>")

}

func (c *Connection) SendPresence(p Presence) (cookie string, err error) {
	// TODO do we need to store the cookie somewhere? present the user with a channel?
	// TODO document that we set the ID
	p.Id = c.getCookie()
	xml.NewEncoder(c).Encode(p)
	return p.Id, nil
	// TODO handle error (both of NewEncoder and what the server will tell us)
}

func (c *Connection) SubscribeStanzas(ch chan<- Stanza) {
	c.subscribers.subscribe(ch)
}
