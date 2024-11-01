package utils

import "strings"

func CleanRepository(repository string) string {
	replacer := strings.NewReplacer(
		"https://", "",
		"https://", "",
		".", "_",
		"/", "_",
	)

	return replacer.Replace(repository)
}
