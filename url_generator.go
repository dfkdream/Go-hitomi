package main

import (
	"path/filepath"
	"strconv"
)

func ImageURLFromImageInfo(info ImageInfo) string {
	h1 := string(info.Hash[len(info.Hash)-1])
	h2 := info.Hash[len(info.Hash)-3 : len(info.Hash)-1]

	var frontendNumber int64 = 3
	subdomain := "a"
	g, err := strconv.ParseInt(h2, 16, 64)
	if err == nil {

		if g < 0x30 {
			frontendNumber = 2
		}

		if g < 0x09 {
			g = 1
		}

		subdomain = string(97+g%frontendNumber) + subdomain
	}

	directory := "images"
	if info.HasWebp == 1 {
		directory = "webp"
	}

	ext := filepath.Ext(info.Name)
	if info.HasWebp == 1 {
		ext = ".webp"
	}

	return "https://" + subdomain + ".hitomi.la/" + directory + "/" + h1 + "/" + h2 + "/" + info.Hash + ext
}
