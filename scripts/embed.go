// Package scripts embeds the browser session export used by installed commands.
package scripts

import _ "embed"

//go:embed export-session.js
var SessionExport string
