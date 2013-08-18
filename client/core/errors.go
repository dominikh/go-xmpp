package core

import (
	"encoding/xml"
)

type ErrBadRequest struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas bad-request"`
	Inner   string   `xml:",chardata"`
}

type ErrConflict struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas conflict"`
	Inner   string   `xml:",chardata"`
}

type ErrFeatureNotImplemented struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas feature-not-implemented"`
	Inner   string   `xml:",chardata"`
}

type ErrForbidden struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas forbidden"`
	Inner   string   `xml:",chardata"`
}

type ErrGone struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas gone"`
	Inner   string   `xml:",chardata"`
}

type ErrInternalServerError struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas internal-server-error"`
	Inner   string   `xml:",chardata"`
}

type ErrItemNotFound struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas item-not-found"`
	Inner   string   `xml:",chardata"`
}

type ErrJIDMalformed struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas jid-malformed"`
	Inner   string   `xml:",chardata"`
}

type ErrNotAcceptable struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas not-acceptable"`
	Inner   string   `xml:",chardata"`
}

type ErrNotAllowed struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas not-allowed"`
	Inner   string   `xml:",chardata"`
}

type ErrNotAuthorized struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas not-authorized"`
	Inner   string   `xml:",chardata"`
}

type ErrPolicyViolation struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas policy-violation"`
	Inner   string   `xml:",chardata"`
}

type ErrRecipientUnavailable struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas recipient-unavailable"`
	Inner   string   `xml:",chardata"`
}

type ErrRedirect struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas redirect"`
	Inner   string   `xml:",chardata"`
}

type ErrRegistrationRequired struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas registration-required"`
	Inner   string   `xml:",chardata"`
}

type ErrRemoteServerNotFound struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas remote-server-not-found"`
	Inner   string   `xml:",chardata"`
}

type ErrRemoteServerTimeout struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas remote-server-timeout"`
	Inner   string   `xml:",chardata"`
}

type ErrResourceConstraint struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas resource-constraint"`
	Inner   string   `xml:",chardata"`
}

type ErrServiceUnavailable struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas service-unavailable"`
	Inner   string   `xml:",chardata"`
}

type ErrSubscriptionRequired struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas subscription-required"`
	Inner   string   `xml:",chardata"`
}

type ErrUndefinedCondition struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas undefined-condition"`
	Inner   string   `xml:",chardata"`
}

type ErrUnexpectedRequest struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas unexpected-request"`
	Inner   string   `xml:",chardata"`
}

func (err ErrBadRequest) Name() xml.Name { return err.XMLName }
func (err ErrBadRequest) Text() string   { return err.Inner }

func (err ErrConflict) Name() xml.Name { return err.XMLName }
func (err ErrConflict) Text() string   { return err.Inner }

func (err ErrFeatureNotImplemented) Name() xml.Name { return err.XMLName }
func (err ErrFeatureNotImplemented) Text() string   { return err.Inner }

func (err ErrForbidden) Name() xml.Name { return err.XMLName }
func (err ErrForbidden) Text() string   { return err.Inner }

func (err ErrGone) Name() xml.Name { return err.XMLName }
func (err ErrGone) Text() string   { return err.Inner }

func (err ErrInternalServerError) Name() xml.Name { return err.XMLName }
func (err ErrInternalServerError) Text() string   { return err.Inner }

func (err ErrItemNotFound) Name() xml.Name { return err.XMLName }
func (err ErrItemNotFound) Text() string   { return err.Inner }

func (err ErrJIDMalformed) Name() xml.Name { return err.XMLName }
func (err ErrJIDMalformed) Text() string   { return err.Inner }

func (err ErrNotAcceptable) Name() xml.Name { return err.XMLName }
func (err ErrNotAcceptable) Text() string   { return err.Inner }

func (err ErrNotAllowed) Name() xml.Name { return err.XMLName }
func (err ErrNotAllowed) Text() string   { return err.Inner }

func (err ErrNotAuthorized) Name() xml.Name { return err.XMLName }
func (err ErrNotAuthorized) Text() string   { return err.Inner }

func (err ErrPolicyViolation) Name() xml.Name { return err.XMLName }
func (err ErrPolicyViolation) Text() string   { return err.Inner }

func (err ErrRecipientUnavailable) Name() xml.Name { return err.XMLName }
func (err ErrRecipientUnavailable) Text() string   { return err.Inner }

func (err ErrRedirect) Name() xml.Name { return err.XMLName }
func (err ErrRedirect) Text() string   { return err.Inner }

func (err ErrRegistrationRequired) Name() xml.Name { return err.XMLName }
func (err ErrRegistrationRequired) Text() string   { return err.Inner }

func (err ErrRemoteServerNotFound) Name() xml.Name { return err.XMLName }
func (err ErrRemoteServerNotFound) Text() string   { return err.Inner }

func (err ErrRemoteServerTimeout) Name() xml.Name { return err.XMLName }
func (err ErrRemoteServerTimeout) Text() string   { return err.Inner }

func (err ErrResourceConstraint) Name() xml.Name { return err.XMLName }
func (err ErrResourceConstraint) Text() string   { return err.Inner }

func (err ErrServiceUnavailable) Name() xml.Name { return err.XMLName }
func (err ErrServiceUnavailable) Text() string   { return err.Inner }

func (err ErrSubscriptionRequired) Name() xml.Name { return err.XMLName }
func (err ErrSubscriptionRequired) Text() string   { return err.Inner }

func (err ErrUndefinedCondition) Name() xml.Name { return err.XMLName }
func (err ErrUndefinedCondition) Text() string   { return err.Inner }

func (err ErrUnexpectedRequest) Name() xml.Name { return err.XMLName }
func (err ErrUnexpectedRequest) Text() string   { return err.Inner }
