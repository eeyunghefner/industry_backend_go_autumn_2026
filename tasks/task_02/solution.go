package main

func rotateRunes(s string, shift int) string {
	sRunes := []rune(s)

	if len(sRunes) == 0 {
		return s
	}

	shift = shift % len(sRunes)

	if shift < 0 {
		shift += len(sRunes)
	}

	sRunes = append(sRunes[shift:], sRunes[:shift]...)
	return string(sRunes)
}
