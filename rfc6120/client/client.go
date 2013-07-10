package client

// TODO make sure whitespace keepalive doesn't break our code
// TODO read messages in a loop
// TODO close connection on a </stream>
// TODO check namespaces everywhere

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"net"
	"strings"
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

type Connection struct {
	net.Conn
	User       string
	Host       string
	decoder    *xml.Decoder
	Features   Features
	Password   string
	cookie     <-chan string
	cookieQuit chan<- struct{}
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

func Connect(user, host, password string) (*Connection, []error) {
	var conn *Connection
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
				}

				break connectLoop
			}
		}
	}

	if conn == nil {
		return nil, errors
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

		if conn.Features.Requires("bind") {
			conn.Bind()
		}
		break
	}
	return conn, errors
}

func (c *Connection) getCookie() string {
	return <-c.cookie
}

func (c *Connection) Bind() {
	// TODO support binding to a user-specified resource
	// TODO handle error cases
	// TODO put IQ sending into its own function
	// TODO use channel callbacks
	fmt.Fprintf(c, "<iq id='%s' type='set'><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/></iq>", c.getCookie())
	c.NextStartElement()
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

		if t, ok := t.(xml.StartElement); ok {
			return &t, nil
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
	fmt.Fprint(c, "</stream:stream>")
	// TODO implement timeout for waiting on </stream> from other end

	// TODO "to help prevent a truncation attack the party that is
	// closing the stream MUST send a TLS close_notify alert and MUST
	// receive a responding close_notify alert from the other party
	// before terminating the underlying TCP connection"
}
