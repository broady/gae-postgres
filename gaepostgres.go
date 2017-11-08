// Copyright 2017 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package gaepostgres is a small wrapper around github.com/lib/pq to provide access to Cloud SQL instances on App Engine standard.
//
// Example usage:
//
//   import (
//   	"database/sql"
//   	"net/http"
//
//   	"google.golang.org/appengine"
//
//   	_ "github.com/broady/gae-postgres"
//   )
//
//  f unc handle(w http.ResponseWriter, r *http.Request) {
//  	 ctx := appengine.NewContext(r)
//
//   	db, err := sql.Open("gae-postgres", "cloudsql=YOUR-INSTANCE-STRING username=postgres password=pw")
//
//   	// ...
//   }
//
// See the pq docs at https://godoc.org/github.comlib/pq for more information on other options for the connection string.
//
// This package also supports an option for the host, ala App Engine flexible paths:
//
//    host=/cloudsql/YOUR-INSTANCE-STRING
package gaepostgres

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net"
	"strings"
	"time"
	"unicode"

	"google.golang.org/appengine/cloudsql"

	"github.com/lib/pq"
)

func init() {
	sql.Register("gae-postgres", aedriver{})
}

type dialer struct {
	instance string
}

func (d dialer) Dial(_, _ string) (net.Conn, error) {
	if true {
	}
	return cloudsql.Dial(d.instance)
}

func (d dialer) DialTimeout(_, _ string, _ time.Duration) (net.Conn, error) {
	return cloudsql.Dial(d.instance)
}

type aedriver struct{}

func (d aedriver) Open(name string) (driver.Conn, error) {
	opts := make(values)
	if err := parseOpts(name, opts); err != nil {
		return nil, err
	}

	if instance, ok := opts["cloudsql"]; ok {
		delete(opts, "cloudsql")
		return pq.DialOpen(dialer{instance}, opts.marshal()+" sslmode=disable")
	}

	if host := opts["host"]; strings.HasPrefix(host, "/cloudsql/") {
		delete(opts, "host")
		instance := host[len("/cloudsql/"):]
		return pq.DialOpen(dialer{instance}, opts.marshal()+" sslmode=disable")
	}

	return pq.Open(name)
}

func (vv values) marshal() string {
	var out string
	for k, v := range vv {
		out += k + "="
		v = strings.Replace(v, `\`, `\\`, -1)
		v = strings.Replace(v, ` `, `\ `, -1)
		v = strings.Replace(v, `'`, `\'`, -1)
		out += v + " "
	}
	return out
}

type values map[string]string

// scanner implements a tokenizer for libpq-style option strings.
type scanner struct {
	s []rune
	i int
}

// newScanner returns a new scanner initialized with the option string s.
func newScanner(s string) *scanner {
	return &scanner{[]rune(s), 0}
}

// Next returns the next rune.
// It returns 0, false if the end of the text has been reached.
func (s *scanner) Next() (rune, bool) {
	if s.i >= len(s.s) {
		return 0, false
	}
	r := s.s[s.i]
	s.i++
	return r, true
}

// SkipSpaces returns the next non-whitespace rune.
// It returns 0, false if the end of the text has been reached.
func (s *scanner) SkipSpaces() (rune, bool) {
	r, ok := s.Next()
	for unicode.IsSpace(r) && ok {
		r, ok = s.Next()
	}
	return r, ok
}

// parseOpts parses the options from name and adds them to the values.
//
// The parsing code is based on conninfo_parse from libpq's fe-connect.c
func parseOpts(name string, o values) error {
	s := newScanner(name)

	for {
		var (
			keyRunes, valRunes []rune
			r                  rune
			ok                 bool
		)

		if r, ok = s.SkipSpaces(); !ok {
			break
		}

		// Scan the key
		for !unicode.IsSpace(r) && r != '=' {
			keyRunes = append(keyRunes, r)
			if r, ok = s.Next(); !ok {
				break
			}
		}

		// Skip any whitespace if we're not at the = yet
		if r != '=' {
			r, ok = s.SkipSpaces()
		}

		// The current character should be =
		if r != '=' || !ok {
			return fmt.Errorf(`missing "=" after %q in connection info string"`, string(keyRunes))
		}

		// Skip any whitespace after the =
		if r, ok = s.SkipSpaces(); !ok {
			// If we reach the end here, the last value is just an empty string as per libpq.
			o[string(keyRunes)] = ""
			break
		}

		if r != '\'' {
			for !unicode.IsSpace(r) {
				if r == '\\' {
					if r, ok = s.Next(); !ok {
						return fmt.Errorf(`missing character after backslash`)
					}
				}
				valRunes = append(valRunes, r)

				if r, ok = s.Next(); !ok {
					break
				}
			}
		} else {
		quote:
			for {
				if r, ok = s.Next(); !ok {
					return fmt.Errorf(`unterminated quoted string literal in connection string`)
				}
				switch r {
				case '\'':
					break quote
				case '\\':
					r, _ = s.Next()
					fallthrough
				default:
					valRunes = append(valRunes, r)
				}
			}
		}

		o[string(keyRunes)] = string(valRunes)
	}

	return nil
}
