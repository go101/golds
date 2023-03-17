package themes

import (
	"os"
	"path/filepath"
)

type Auto struct{}

func (Auto) Name() string { return "auto" }

func (Auto) CSS() string { return auto_css }

var auto_css = light_css // default to light_css

func init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}

	data, err := os.ReadFile(filepath.Join(configDir, "golds", "custom.css"))
	if err != nil {
		return
	}

	auto_css += string(data)
}
