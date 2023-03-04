package theme

import (
	"embed"
	"fmt"
)

//go:embed assets
var Assets embed.FS

func ThemeCSS(themeName string) (string, error) {
	fname := fmt.Sprintf("assets/%[1]s/%[1]s.css", themeName)
	bsCSS, err := Assets.ReadFile(fname)
	return string(bsCSS), err
}
