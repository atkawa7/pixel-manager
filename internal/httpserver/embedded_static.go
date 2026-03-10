package httpserver

import "embed"

//go:embed all:public
var embeddedPublicFiles embed.FS
