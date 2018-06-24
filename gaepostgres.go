// Copyright 2017 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package gaepostgres is a small wrapper around github.com/lib/pq to provide access to Cloud SQL instances on App Engine standard.
//
// Example usage:
//
//	import (
//		"database/sql"
//		"net/http"
//
//		_ "github.com/broady/gae-postgres"
//		"google.golang.org/appengine"
//	)
//
//	func handle(w http.ResponseWriter, r *http.Request) {
//		db, err := sql.Open("gae-postgres", "cloudsql=YOUR-INSTANCE-STRING username=postgres password=pw")
//
//		// ...
//	}
//
// See the pq docs at https://godoc.org/github.comlib/pq for more information on other options for the connection string.
//
// This package also supports an option for the host, ala App Engine flexible paths:
//
//	host=/cloudsql/YOUR-INSTANCE-STRING
package gaepostgres

import (
	"database/sql"
	"database/sql/driver"
	"net"
	"strings"
	"time"

	"github.com/lib/pq"
	"google.golang.org/appengine/cloudsql"
)

func init() {
	sql.Register("gae-postgres", aedriver{})
}

type dialer struct {
	instance string
}

func (d dialer) Dial(_, _ string) (net.Conn, error) {
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
