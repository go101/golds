package theme

type Light struct{}

func (*Light) Name() string { return "light" }

func (*Light) CSS() string {
	bsCSS, err := ThemeCSS("light")
	if err != nil {
		panic(err)
	}
	return string(bsCSS)
}
