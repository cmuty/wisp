package i18n

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

//go:embed ru.json
var russianJSON []byte
var russian = func() map[string]string {
	var messages map[string]string
	if err := json.Unmarshal(russianJSON, &messages); err != nil {
		panic(err)
	}
	return messages
}()
var formatToken = regexp.MustCompile(`%%|%[-+0-9.#]*[sdvqw]`)

type pattern struct {
	re          *regexp.Regexp
	translation string
	tokens      []string
}

var patterns = func() []pattern {
	keys := make([]string, 0, len(russian))
	for key := range russian {
		if strings.Contains(key, "%") {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) == len(keys[j]) {
			return keys[i] < keys[j]
		}
		return len(keys[i]) > len(keys[j])
	})
	var result []pattern
	for _, key := range keys {
		parts := formatToken.FindAllStringIndex(key, -1)
		if len(parts) == 0 {
			continue
		}
		expression := "^"
		previous := 0
		var tokens []string
		for _, span := range parts {
			expression += regexp.QuoteMeta(key[previous:span[0]])
			token := key[span[0]:span[1]]
			if token == "%%" {
				expression += "%"
			} else {
				if strings.HasSuffix(token, "d") {
					expression += "(-?[0-9]+)"
				} else {
					expression += "(.*?)"
				}
				tokens = append(tokens, token)
			}
			previous = span[1]
		}
		expression += regexp.QuoteMeta(key[previous:]) + "$"
		result = append(result, pattern{regexp.MustCompile(expression), russian[key], tokens})
	}
	return result
}()

func Normalize(language string) string {
	if language == "ru" {
		return "ru"
	}
	return "en"
}
func Text(language, text string) string {
	if Normalize(language) != "ru" {
		return text
	}
	return translate(text, 0)
}
func translate(text string, depth int) string {
	if depth > 6 {
		return text
	}
	if translated, ok := russian[text]; ok {
		return translated
	}
	if strings.Contains(text, "\n") {
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			lines[i] = translate(line, depth+1)
		}
		return strings.Join(lines, "\n")
	}
	for _, entry := range patterns {
		matched := entry.re.FindStringSubmatch(text)
		if matched == nil {
			continue
		}
		index := 0
		return formatToken.ReplaceAllStringFunc(entry.translation, func(token string) string {
			if token == "%%" {
				return "%"
			}
			index++
			if index >= len(matched) {
				return token
			}
			value := matched[index]
			if index <= len(entry.tokens) && strings.HasSuffix(entry.tokens[index-1], "w") {
				value = translate(value, depth+1)
			}
			return value
		})
	}
	for _, prefix := range []string{"SoundCloud: ", "Spotify: ", "YouTube Music: ", "soundcloud: ", "spotify: ", "youtube: "} {
		if strings.HasPrefix(text, prefix) {
			return prefix + translate(strings.TrimPrefix(text, prefix), depth+1)
		}
	}

	return text
}
