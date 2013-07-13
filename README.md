# XMPP

## Current state of the library

Don't use it yet, please.

Even though most parts of RFC 6120 and RFC 6121, as well as some XEPs,
have been implemented already, there are a lot of things missing or in
flux.

For one, the API is far from stable. There are regular changes to the
API, changing types and function signatures until an optimal API has
been found.

Furthermore, there's little error handling yet. Most code expects
things to work and will fail in unexpected ways if errors occur.

And last, there's no documentation or examples yet, either.

## Design goals

Unlike some of the existing XMPP libraries for Go, this one strives to
cleanly separate the different specifications and extensions. On the
lowest level, there's an RFC 6120 client, which is the plain XMPP
protocol that specifies how to transmit stanzas. One level up, there's
an RFC 6121 client, which adds XMPP IM capabilities.

Further functionality is bundled in XMPP extensions (XEPs) which can
be implemented and used independently of each other (unless XEPs
explicitly depend on each other) and used with either the RFC 6120 or
RFC 6121 clients, again depending on specific requirements.

It would even be possible to implement your own non-standard
functionality on top of the plain XMPP protocol without having IM
related functionality get in your way.

## Scope

Initially, this package will only help with implementing clients. It
is, however, planned to also add server capabilities and the package
layout has been chosen with that in mind, putting code in either a
`client`, `server` or `shared` path. Due to the current state of the
library (see above), however, a lot of code that should be shared is
still in the client package. This will change before the first
release.

## License

Copyright (c) 2013 Dominik Honnef

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
