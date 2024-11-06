package utils

import "strings"

func CleanRepository(repository string) string {
	replacer := strings.NewReplacer(
		"https://", "",
		"https://", "",
		"/", "-",
	)

	return replacer.Replace(repository)
}
