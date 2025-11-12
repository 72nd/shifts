package export

import (
	"embed"
)

//go:embed static
var static embed.FS

type TypstExport struct {
}
