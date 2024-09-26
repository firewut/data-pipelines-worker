package config

import "embed"

//go:embed assets/fonts/*.ttf
var FontFiles embed.FS

func LoadFont(fontName string) ([]byte, error) {
	return FontFiles.ReadFile("assets/fonts/" + fontName)
}

func ListFonts() ([]string, error) {
	entries, err := FontFiles.ReadDir("assets/fonts")
	if err != nil {
		return nil, err
	}

	var fontNames []string
	for _, entry := range entries {
		if !entry.IsDir() {
			fontNames = append(fontNames, entry.Name())
		}
	}
	return fontNames, nil
}
