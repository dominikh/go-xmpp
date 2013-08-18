package core

import (
	"encoding/xml"
)

type Feature interface {
	Name() string
	Required() bool
}

type StartTLS struct {
	required bool
}

func (StartTLS) Name() string {
	return "starttls"
}

func (f StartTLS) Required() bool {
	return f.required
}

type UnsupportedFeature struct {
	name string
}

func (f UnsupportedFeature) Name() string {
	return f.name
}

func (UnsupportedFeature) Required() bool {
	// TODO reconsider the decision to return false
	return false
}

type Bind struct{}

func (Bind) Name() string {
	return "bind"
}

func (Bind) Required() bool {
	return true
}

type SASL []string // TODO consider using a Mechanism struct for
// mechanisms that have additional data

func (SASL) Required() bool {
	return true
}

func (SASL) Name() string {
	return "sasl"
}

type Features map[string]Feature

func (fs Features) Requires(name string) bool {
	if f, ok := fs[name]; ok {
		return f.Required()
	}

	return false
}

func (fs Features) Includes(name string) bool {
	_, ok := fs[name]
	return ok
}

func (fs Features) RequiresTLS() bool {
	if f, ok := fs["starttls"]; ok {
		return len(fs) == 1 || f.Required()
	}

	return false
}

func (c *Conn) parseFeatures() error {
	features := make(Features)

	_, err := c.nextStartElement() // FIXME flow. this skips over the stream
	if err != nil {
		return err
	}

	for {
		// TODO handle not getting to the end of the features (connection timeout?)
		t, err := c.nextToken()
		if err != nil {
			return err
		}
		if t, ok := t.(xml.StartElement); ok {
			// FIXME namespace
			switch t.Name.Local {
			case "starttls":
				var f struct {
					Required xml.Name `xml:"required"`
				}
				err = c.decoder.DecodeElement(&f, &t)
				if err != nil {
					return err
				}
				features["starttls"] = StartTLS{f.Required.Local != ""}
			case "bind":
				features["bind"] = Bind{}
				c.decoder.Skip()
			case "mechanisms":
				var f struct {
					Mechanisms []struct {
						Name string `xml:",chardata"`
					} `xml:"mechanism"`
				}
				err = c.decoder.DecodeElement(&f, &t)
				if err != nil {
					return err
				}
				mechanisms := make(SASL, len(f.Mechanisms))
				for i, m := range f.Mechanisms {
					mechanisms[i] = m.Name
				}
				features["sasl"] = mechanisms
			default:
				features[t.Name.Local] = UnsupportedFeature{t.Name.Local}
				c.decoder.Skip()

			}
		} else {
			break
		}
	}

	c.features = features
	return nil
}

func (c *Conn) Features() Features {
	return c.features
}
