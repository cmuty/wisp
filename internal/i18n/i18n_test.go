package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCatalogPreservesFormatValues(t *testing.T) {
	cases := map[string]string{
		"Search: Welcome to wisp.":                                             "Поиск: Welcome to wisp.",
		"Added 12 tracks to your queue. /play to begin.":                       "В очередь добавлено треков: 12. Начать: /play.",
		"Could not save library: permission denied":                            "Не удалось сохранить медиатеку: доступ запрещён",
		"SoundCloud: soundcloud: token expired or invalid; supply a new token": "SoundCloud: soundcloud: токен недействителен или истёк; добавь новый",
		"Unknown command /hello. Type /help.":                                  "Неизвестная команда /hello. Введи /help.",
	}
	for en, want := range cases {
		if got := Text("ru", en); got != want {
			t.Errorf("%q -> %q, want %q", en, got, want)
		}
		if Text("en", en) != en {
			t.Fatal("English changed")
		}
	}
	for en, ru := range russian {
		a, b := formatToken.FindAllString(en, -1), formatToken.FindAllString(ru, -1)
		if strings.Join(a, ",") != strings.Join(b, ",") {
			t.Errorf("format differs: %q -> %q", en, ru)
		}
	}
}
func TestOwnedErrorsHaveRussianTranslations(t *testing.T) {
	for _, folder := range []string{"source", "lyrics", "library", "player", "command"} {
		files, err := os.ReadDir(filepath.Join("..", folder))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".go") || strings.HasSuffix(file.Name(), "_test.go") {
				continue
			}
			tree, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", folder, file.Name()), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(tree, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				owner, ok := selector.X.(*ast.Ident)
				if !ok || !(owner.Name == "errors" && selector.Sel.Name == "New" || owner.Name == "fmt" && selector.Sel.Name == "Errorf") {
					return true
				}
				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				text, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				if _, ok := russian[text]; !ok && text != "mpv: %s" {
					t.Errorf("missing error translation %s/%s: %q", folder, file.Name(), text)
				}
				return true
			})
		}
	}
}
