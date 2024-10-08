package helpers

import (
	"fmt"
	"log"
)

func HexToRGB(hex string) (uint8, uint8, uint8) {
	var r, g, b uint8
	if len(hex) == 7 {
		_, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("Invalid hex color format")
	}
	return r, g, b
}
