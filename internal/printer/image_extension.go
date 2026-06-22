package printer

func detectImageExtension(data []byte) string {
	if len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return ".png"
	}

	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ".jpg"
	}

	if len(data) >= 6 {
		header := string(data[:6])
		if header == "GIF87a" || header == "GIF89a" {
			return ".gif"
		}
	}

	if len(data) >= 2 && data[0] == 0x42 && data[1] == 0x4D {
		return ".bmp"
	}

	return ".png"
}
