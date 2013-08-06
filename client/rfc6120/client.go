package rfc6120

// TODO make sure whitespace keepalive doesn't break our code
// TODO check namespaces everywhere
// TODO optional reconnect handling: 1) reconnect if enabled 2) close
// channels when the connection is gone for good
// TODO add a namespace registry, and send <service-unavailable/>
// errors for unsupported namespaces (section 8.4)

import (
	shared "honnef.co/go/xmpp/shared/rfc6120"
	"honnef.co/go/xmpp/shared/xep"

	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
)

var _ Client = &Connection{}

const (
	nsStream  = "http://etherx.jabber.org/streams"
	nsTLS     = "urn:ietf:params:xml:ns:xmpp-tls"
	nsSASL    = "urn:ietf:params:xml:ns:xmpp-sasl"
	nsBind    = "urn:ietf:params:xml:ns:xmpp-bind"
	nsSession = "urn:ietf:params:xml:ns:xmpp-session"
	nsClient  = "jabber:client"
)

var SupportedMechanisms = []string{"PLAIN"}
var ErrorTypes = make(map[xml.Name]XMPPError)
var XEPs = make(map[int]func(Client) error)

func init() {
	// We're using RegisterErrorType instead of directly populating
	// the map to make use of its checks for correctness.
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "bad-request", ErrBadRequest{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "conflict", ErrConflict{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "feature-not-implemented", ErrFeatureNotImplemented{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "forbidden", ErrForbidden{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "gone", ErrGone{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "internal-server-error", ErrInternalServerError{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "item-not-found", ErrItemNotFound{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "jid-malformed", ErrJIDMalformed{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "not-acceptable", ErrNotAcceptable{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "not-allowed", ErrNotAllowed{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "not-authorized", ErrNotAuthorized{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "policy-violation", ErrPolicyViolation{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "recipient-unavailable", ErrRecipientUnavailable{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "redirect", ErrRedirect{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "registration-required", ErrRegistrationRequired{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "remote-server-not-found", ErrRemoteServerNotFound{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "remote-server-timeout", ErrRemoteServerTimeout{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "resource-constraint", ErrResourceConstraint{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "service-unavailable", ErrServiceUnavailable{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "subscription-required", ErrSubscriptionRequired{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "undefined-condition", ErrUndefinedCondition{})
	RegisterErrorType("urn:ietf:params:xml:ns:xmpp-stanzas", "unexpected-request", ErrUnexpectedRequest{})
}

type XMPPError interface {
	Name() xml.Name
	Text() string
}

// RegisterErrorType is used to register errors that are not covered
// by the core specification, like they are used by various XEPs.
//
// Errors are specified by their namespace, tag name and a struct to
// hold additional information.
//
// This function is not thread-safe. It is advised to call it from a
// package's init function. Trying to register the same error twice
// will panic. The provided error must not be a pointer.
func RegisterErrorType(space, local string, err XMPPError) {
	if reflect.ValueOf(err).Kind() == reflect.Ptr {
		panic("Must not call RegisterErrorType with pointer type")
	}

	name := xml.Name{Space: space, Local: local}
	if _, ok := ErrorTypes[name]; ok {
		panic(fmt.Sprintf("An error type for '%s %s' has already been registered", space, local))
	}

	ErrorTypes[name] = err
}

func RegisterXEP(n int, fn func(Client) error) {
	if _, ok := XEPs[n]; ok {
		panic(fmt.Sprintf("XEP %d has already been registered", n))
	}

	XEPs[n] = fn
}

type Client interface {
	io.Writer
	SendIQ(to, typ string, value interface{}) (chan *IQ, string)
	SendIQReply(iq *IQ, typ string, value interface{})
	SendPresence(p Presence) (cookie string, err error)
	EmitStanza(s Stanza)
	SubscribeStanzas(ch chan<- Stanza)
	JID() string
	Features() Features
	Close()

	// RegisterXEP registers a XEP and reports success. If any of the
	// dependencies haven't been registered, false will be returned.
	RegisterXEP(n int, x xep.Interface, required ...int) error

	// GetXEP tries to return a registered XEP.
	GetXEP(n int) (xep.Interface, bool)

	// MustGetXEP behaves like GetXEP but panics if the XEP hasn't
	// been registered.
	MustGetXEP(n int) xep.Interface
}

func Resolve(host string) ([]shared.Address, []error) {
	return shared.ResolveFQDN(host, "xmpp-client")
}

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

func (s *subscribers) send(stanza Stanza) (delivered bool) {
	s.RLock()
	defer s.RUnlock()

	toSkip := len(s.chans)
	for _, ch := range s.chans {
		select {
		case ch <- stanza:
		default:
			toSkip--
		}
	}

	// if toSkip == 0, none of the subscribers were able to receive the stanza, so
	// we definitely couldn't process it
	return toSkip != 0
}

func (s *subscribers) subscribe(ch chan<- Stanza) {
	s.Lock()
	defer s.Unlock()
	s.chans = append(s.chans, ch)
}

type Connection struct {
	net.Conn
	extensions *extensions
	mutex      sync.Mutex
	user       string
	host       string
	decoder    *xml.Decoder
	features   Features
	password   string
	cookie     <-chan string
	cookieQuit chan<- struct{}
	jid        string
	callbacks  map[string]chan *IQ
	closing    bool
	// TODO reconsider choice of structure when we allow unsubscribing
	subscribers subscribers
}

type extensions struct {
	sync.RWMutex
	m map[int]xep.Interface
}

func (e *extensions) get(n int) (xep.Interface, bool) {
	e.RLock()
	defer e.RUnlock()
	x, ok := e.m[n]
	return x, ok
}

func (e *extensions) set(n int, x xep.Interface) {
	e.Lock()
	defer e.Unlock()
	e.m[n] = x
}

type DependencyError struct {
	XEP     int
	Missing int
}

func (d DependencyError) Error() string {
	return fmt.Sprintf("Could not register XEP-%d because the dependency XEP-%d could not be found",
		d.XEP, d.Missing)
}

func (c *Connection) RegisterXEP(n int, x xep.Interface, required ...int) error {
	for _, req := range required {
		if _, ok := c.extensions.get(req); !ok {
			if wrapper, ok := XEPs[req]; ok {
				wrapper(c)
			}
			return DependencyError{n, req}
		}
	}

	c.extensions.set(n, x)
	return nil
}

func (c *Connection) GetXEP(n int) (xep.Interface, bool) {
	c.extensions.RLock()
	defer c.extensions.RUnlock()

	x, ok := c.extensions.m[n]

	return x, ok
}

func (c *Connection) MustGetXEP(n int) xep.Interface {
	x, ok := c.GetXEP(n)
	if !ok {
		panic(fmt.Sprintf("XEP-%04d is not registered", n))
	}

	return x
}

func (c *Connection) EmitStanza(stanza Stanza) {
	c.subscribers.send(stanza)
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

func newConnection(c net.Conn) *Connection {
	cookieChan := make(chan string)
	cookieQuitChan := make(chan struct{})
	go generateCookies(cookieChan, cookieQuitChan)
	return &Connection{
		Conn:       c,
		decoder:    xml.NewDecoder(c),
		cookie:     cookieChan,
		cookieQuit: cookieQuitChan,
		callbacks:  make(map[string]chan *IQ),
		extensions: &extensions{m: make(map[int]xep.Interface)},
	}

}

func Dial(user, host, password string) (client Client, errors []error, ok bool) {
	var conn *Connection
	addrs, errors := Resolve(host)

connectLoop:
	for _, addr := range addrs {
		for _, ip := range addr.IPs {
			c, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: ip, Port: addr.Port})
			if err != nil {
				errors = append(errors, err)
				continue
			}

			conn = newConnection(c)
			conn.host = host
			conn.user = user
			conn.password = password
			break connectLoop
		}
	}

	if conn == nil {
		return nil, errors, false
	}

	moreErrors := conn.setUp()
	errors = append(errors, moreErrors...)

	return conn, errors, true
}

// DialOn works like Dial but expects an existing and open net.Conn to
// use.
func DialOn(c net.Conn, user, host, password string) (client Client, errors []error, ok bool) {
	conn := newConnection(c)
	conn.host = host
	conn.user = user
	conn.password = password

	errors = conn.setUp()

	return conn, errors, len(errors) == 0
}

func (conn *Connection) setUp() []error {
	// TODO error handling
	for {
		conn.openStream()
		conn.receiveStream()
		conn.parseFeatures()
		if conn.features.Includes("starttls") {
			conn.startTLS() // TODO handle error
			continue
		}

		if conn.features.Requires("sasl") {
			conn.sasl()
			continue
		}
		break
	}

	go conn.read()
	conn.bind()

	return nil
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

	Subject string `xml:"subject,omitempty"`
	Body    string `xml:"body"` // TODO omitempty?
	Error   *Error `xml:"error,omitempty"`
	Thread  string `xml:"thread,omitempty"`
	Inner   []byte `xml:",innerxml"`
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
	Inner    []byte `xml:",innerxml"`
}

func (p Presence) IsError() bool {
	return p.Error != nil
}

type IQ struct { // info/query
	XMLName xml.Name `xml:"jabber:client iq"`
	Header

	Error *Error   `xml:"error"`
	Query xml.Name `xml:"query"`
	Inner []byte   `xml:",innerxml"`
}

func (iq IQ) IsError() bool {
	return iq.Error != nil
}

type Error struct {
	XMLName  xml.Name `xml:"jabber:client error"`
	Type     string   `xml:"type,attr"`
	Text     string   `xml:"text"` // TODO do we need to specify the namespace here?
	InnerXML []byte   `xml:",innerxml"`
}

func (err Error) Error() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("(%s) ", err.Type))
	for _, e := range err.Errors() {
		b.WriteString(fmt.Sprintf("<%s/>", e.Name().Local))
	}

	return b.String()
}

func (err Error) Errors() []XMPPError {
	var errors []XMPPError

	r := bytes.NewReader(err.InnerXML)
	dec := xml.NewDecoder(r)

	for {
		// TODO handle error
		t, err := dec.Token()
		if err == io.EOF {
			break
		}

		if start, ok := t.(xml.StartElement); ok {
			errType, ok := ErrorTypes[start.Name]
			if !ok {
				// TODO stuff it into an "unrecognized error" struct or something
				continue
			}
			errValue := reflect.New(reflect.TypeOf(errType)).Interface()
			// TODO handle error
			dec.DecodeElement(errValue, &start)
			errors = append(errors, errValue.(XMPPError))
		}
	}

	return errors
}

type streamError struct {
	XMLName xml.Name `xml:"http://etherx.jabber.org/streams error"`
	Any     xml.Name `xml:",any"`
	Text    string   `xml:"text"`
}

func (streamError) ID() string {
	return ""
}

func (streamError) IsError() bool {
	return true
}

func (c *Connection) JID() string {
	return c.jid
}

func (c *Connection) read() {
	for {
		t, _ := c.nextStartElement()

		if t == nil {
			c.mutex.Lock()
			for _, ch := range c.callbacks {
				close(ch)
			}
			c.mutex.Unlock()
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
		default:
			fmt.Println(t.Name.Local)
			// TODO handle error
		}

		// Unmarshal into that storage.
		c.decoder.DecodeElement(nv, t)
		// TODO what about message and presence? They can return
		// errors, too, but they don't have any ID associated with
		// them. how do we want to present such kinds of errors to the
		// user?
		if iq, ok := nv.(*IQ); ok && (iq.Type == "result" || iq.Type == "error") {
			c.mutex.Lock()
			if ch, ok := c.callbacks[nv.ID()]; ok {
				ch <- iq
				delete(c.callbacks, nv.ID())
			}
			c.mutex.Unlock()
		} else {
			delivered := c.subscribers.send(nv)
			if !delivered {
				c.SendError(nv, "wait", "", ErrResourceConstraint{})
			}
		}
	}
}

func (c *Connection) getCookie() string {
	return <-c.cookie
}

func (c *Connection) bind() {
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
	c.jid = bind.JID
}

func (c *Connection) reset() {
	c.decoder = xml.NewDecoder(c.Conn)
	c.features = nil
}

func (c *Connection) sasl() {
	payload := fmt.Sprintf("\x00%s\x00%s", c.user, c.password)
	payloadb64 := base64.StdEncoding.EncodeToString([]byte(payload))
	fmt.Fprintf(c, "<auth xmlns='urn:ietf:params:xml:ns:xmpp-sasl' mechanism='PLAIN'>%s</auth>", payloadb64)
	t, _ := c.nextStartElement() // FIXME error handling
	if t.Name.Local == "success" {
		c.reset()
	} else {
		// TODO handle the error case
	}

	// TODO actually determine which mechanism we can use, use interfaces etc to call it
}

func (c *Connection) startTLS() error {
	fmt.Fprint(c, "<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>")
	t, _ := c.nextStartElement() // FIXME error handling
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

	if err := tlsConn.VerifyHostname(c.host); err != nil {
		return errors.New("xmpp: failed to match TLS certificate to name: " + err.Error()) // FIXME
	}

	c.Conn = tlsConn
	c.reset()

	return nil
}

// TODO Move this outside of client. This function will be used by
// servers, too.
func (c *Connection) nextStartElement() (*xml.StartElement, error) {
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

func (c *Connection) nextToken() (xml.Token, error) {
	return c.decoder.Token()
}

type UnexpectedMessage struct {
	Name string
}

func (e UnexpectedMessage) Error() string {
	return e.Name
}

// TODO return error of Fprintf
func (c *Connection) openStream() {
	// TODO consider not including the JID if the connection isn't encrypted yet
	// TODO configurable xml:lang
	fmt.Fprintf(c, "<?xml version='1.0' encoding='UTF-8'?><stream:stream from='%s@%s' to='%s' version='1.0' xml:lang='en' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>",
		c.user, c.host, c.host)
}

type UnsupportedVersion struct {
	Version string
}

func (e UnsupportedVersion) Error() string {
	return "Unsupported XMPP version: " + e.Version
}

func (c *Connection) receiveStream() error {
	t, err := c.nextStartElement() // TODO error handling
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
	if c.closing {
		// Terminate TCP connection
		c.Conn.Close()
		return
	}

	fmt.Fprint(c, "</stream:stream>")
	c.closing = true
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
	buf := &bytes.Buffer{}

	cookie := c.getCookie()
	reply := make(chan *IQ, 1)
	c.mutex.Lock()
	c.callbacks[cookie] = reply
	c.mutex.Unlock()

	toAttr := ""
	if len(to) > 0 {
		toAttr = "to='" + xmlEscape(to) + "'"
	}

	fmt.Fprintf(buf, "<iq %s from='%s' type='%s' id='%s'>", toAttr, xmlEscape(c.jid), xmlEscape(typ), cookie)
	xml.NewEncoder(buf).Encode(value)
	fmt.Fprintf(buf, "</iq>")

	io.Copy(c, buf)

	return reply, cookie
}

// TODO get rid of to and id arguments, use IQ value instead
func (c *Connection) SendIQReply(iq *IQ, typ string, value interface{}) {
	toAttr := ""
	if len(iq.From) > 0 {
		toAttr = "to='" + xmlEscape(iq.From) + "'"
	}

	fmt.Fprintf(c, "<iq %s from='%s' type='%s' id='%s'>", toAttr, xmlEscape(c.jid), xmlEscape(typ), iq.Id)
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

// TODO reconsider name, since it conflicts with the idea of sending
// stream errors as opposed to stanza errors
func (c *Connection) SendError(inReplyTo Stanza, typ string, text string, errors ...XMPPError) {
	if inReplyTo.IsError() {
		// 8.3.1: An entity that receives an error stanza MUST NOT
		// respond to the stanza with a further error stanza; this
		// helps to prevent looping.
		return
	}
	var tag, id, from, to string
	id = inReplyTo.ID()

	switch t := inReplyTo.(type) {
	case *Message:
		tag = "message"
		from = t.From
		to = t.To
	case *Presence:
		tag = "presence"
		from = t.From
		to = t.To
	case *IQ:
		tag = "iq"
		from = t.From
		to = t.To
	default:
		// TODO what to do here?
		return
	}

	buf := &bytes.Buffer{}

	if id != "" {
		fmt.Fprintf(buf, "<%s from='%s' to='%s' id='%s' type='error'>", tag, to, from, id) // We swap to and from
	} else {
		fmt.Fprintf(buf, "<%s from='%s' to='%s' type='error'>", tag, to, from) // We swap to and from
	}
	fmt.Fprintf(buf, "<error type='%s'>", typ)
	enc := xml.NewEncoder(buf)
	for _, error := range errors {
		enc.Encode(error) // TODO handle error
	}
	fmt.Fprintf(buf, "</error></%s>", tag)
	io.Copy(c, buf)
}

func (c *Connection) SubscribeStanzas(ch chan<- Stanza) {
	c.subscribers.subscribe(ch)
}
