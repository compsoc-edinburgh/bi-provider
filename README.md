# bi-provider [![Go Report Card](https://goreportcard.com/badge/github.com/compsoc-edinburgh/bi-provider)](https://goreportcard.com/report/github.com/compsoc-edinburgh/bi-provider)

This is the API for Better Informatics that provides information about the currently logged in user.

## How does it get my details?

When you log in on Better Informatics, Informatics login server (CoSign) sends Better Informatics a personalised service-only (i.e, only valid to us) login cookie. We then store this login cookie as a cookie on your machine, available to all subdomains of _betterinformatics.com_.

We can see that login cookie, so we take it, and ask Informatics what your UUN is. Then we use Informatics' LDAP to get your details.

## Does this mean other people can see my personal information?

Absolutely not. We set the `Access-Control-Allow-Origin` header, causing your browser to reject API calls sent from websites other than `https://betterinformatics.com`.

Found a security vulnerability? Please email [qaisjp](mailto:me@qaisjp.com) right away.

## What's left?

- Using LDAP to get user information

## CoSign?

This repository uses [gosign](https://github.com/qaisjp/gosign), a library to interact with CoSign daemons.

It also interacts with the [cosign-webapi](https://github.com/qaisjp/cosign-webapi) backend, available to Better Informatics services on [TARDIS](https://wiki.tardis.ed.ac.uk).
