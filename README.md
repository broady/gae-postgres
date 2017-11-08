# Cloud SQL for PostgreSQL on Google App Engine

[GoDoc](https://godoc.org/github.com/broady/gae-postgres)

## Example

app.yaml

    runtime: go
    api_version: go1

    handlers:
    - url: /.*
      script: _go_app

    env_variables:
      # Replace INSTANCE_CONNECTION_NAME with the value obtained when configuring your
      # Cloud SQL instance, available from the Google Cloud Console or from the Cloud SDK.
      # For Cloud SQL 2nd generation instances, this should be in the form of "project:region:instance".
      CLOUDSQL_CONNECTION_NAME: 'INSTANCE_CONNECTION_NAME'
      # Replace username and password if you aren't using the root user.
      CLOUDSQL_USER: postgres
      CLOUDSQL_PASSWORD: pw

main.go

    // Copyright 2016 Google Inc. All rights reserved.
    // Use of this source code is governed by the Apache 2.0
    // license that can be found in the LICENSE file.

    // Sample cloudsql demonstrates connection to a Postgres Cloud SQL instance from App Engine standard.
    package cloudsql

    import (
      "bytes"
      "database/sql"
      "fmt"
      "log"
      "net/http"
      "os"

      _ "github.com/broady/gae-postgres"
    )

    func init() {
      http.HandleFunc("/", handler)
    }

    func handler(w http.ResponseWriter, r *http.Request) {
      if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
      }

      connectionName := mustGetenv("CLOUDSQL_CONNECTION_NAME")
      user := mustGetenv("CLOUDSQL_USER")
      password := os.Getenv("CLOUDSQL_PASSWORD") // NOTE: password may be empty

      w.Header().Set("Content-Type", "text/plain")

      db, err := sql.Open("gae-postgres", fmt.Sprintf("user=%s password=%s cloudsql=%s", user, password, connectionName))
      if err != nil {
        http.Error(w, fmt.Sprintf("Could not open db: %v", err), 500)
        return
      }
      defer db.Close()

      rows, err := db.Query("SELECT datname FROM pg_database")
      if err != nil {
        http.Error(w, fmt.Sprintf("Could not query db: %v", err), 500)
        return
      }
      defer rows.Close()

      buf := bytes.NewBufferString("Databases:\n")
      for rows.Next() {
        var dbName string
        if err := rows.Scan(&dbName); err != nil {
          http.Error(w, fmt.Sprintf("Could not scan result: %v", err), 500)
          return
        }
        fmt.Fprintf(buf, "- %s\n", dbName)
      }
      w.Write(buf.Bytes())
    }

    func mustGetenv(k string) string {
      v := os.Getenv(k)
      if v == "" {
        log.Panicf("%s environment variable not set.", k)
      }
      return v
    }
