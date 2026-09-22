// Package web embeds the HTML templates and static assets directly into
// the compiled binary, so the server doesn't depend on the working
// directory at runtime — it works identically whether you run it from
// the project root locally, or a hosting platform runs the binary from
// some other directory (as some serverless/PaaS environments do).
package web

import "embed"

//go:embed templates/*.html
var Templates embed.FS

//go:embed static
var Static embed.FS
