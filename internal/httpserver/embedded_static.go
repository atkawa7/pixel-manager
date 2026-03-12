package httpserver

import "embed"

//go:embed all:public
var embeddedPublicFiles embed.FS

//go:embed all:openapi
var embeddedOpenAPIFiles embed.FS
