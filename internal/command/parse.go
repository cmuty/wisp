package command

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

type Input struct {
	Name string
	Args []string
	Text string
}

// Backslashes stay literal so Windows file paths do not need shell escaping.
// Double the enclosing quote to include that character inside a quoted value.
func Parse(line string) (Input, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Input{}, nil
	}
	if !strings.HasPrefix(line, "/") {
		if u, e := url.Parse(line); e == nil && u.Scheme == "https" && u.Host != "" {
			return Input{Name: "import", Args: []string{line}}, nil
		}
		return Input{Name: "search", Text: line, Args: []string{line}}, nil
	}
	var words []string
	var b strings.Builder
	var quote rune
	started := false
	rs := []rune(line[1:])
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if quote != 0 {
			if r == quote {
				if i+1 < len(rs) && rs[i+1] == quote {
					b.WriteRune(r)
					i++
				} else {
					quote = 0
				}
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
		} else if unicode.IsSpace(r) {
			if started {
				words = append(words, b.String())
				b.Reset()
				started = false
			}
		} else {
			b.WriteRune(r)
			started = true
		}
	}
	if quote != 0 {
		return Input{}, errors.New("close the quoted value")
	}
	if started {
		words = append(words, b.String())
	}
	if len(words) == 0 {
		return Input{}, errors.New("type a command after /")
	}
	return Input{Name: strings.ToLower(words[0]), Args: words[1:], Text: strings.Join(words[1:], " ")}, nil
}
