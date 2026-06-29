package cmd

import _ "embed"

//go:embed migrations.sql
var migrationsSQL string
