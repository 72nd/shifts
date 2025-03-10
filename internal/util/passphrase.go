package util

import (
	"bufio"
	"embed"
	"fmt"
	"math/rand"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed static/adjectives.txt static/nouns.txt
var wordlists embed.FS

var Passphrase = Generator{
	Adjectives: []string{},
	Nouns:      []string{},
}

type Generator struct {
	Adjectives []string
	Nouns      []string
}

var PwdGenerator = Generator{}

func (g *Generator) Generate() (string, error) {
	if err := g.ensureLoaded(); err != nil {
		return "", err
	}
	adj1 := g.Adjectives[rand.Intn(len(g.Adjectives))]
	adj2 := g.Adjectives[rand.Intn(len(g.Adjectives))]
	noun := g.Nouns[rand.Intn(len(g.Nouns))]
	rsl := fmt.Sprintf(
		"%s%s%s%02d",
		cases.Title(language.English, cases.NoLower).String(adj1),
		cases.Title(language.English, cases.NoLower).String(adj2),
		cases.Title(language.English, cases.NoLower).String(noun),
		rand.Intn(100),
	)
	return rsl, nil
}

func (g *Generator) ensureLoaded() error {
	if len(g.Adjectives) == 0 {
		adjectives, err := loadWordlist("static/adjectives.txt")
		if err != nil {
			return err
		}
		g.Adjectives = adjectives
	}
	if len(g.Nouns) == 0 {
		nouns, err := loadWordlist("static/nouns.txt")
		if err != nil {
			return err
		}
		g.Nouns = nouns
	}
	return nil
}

func loadWordlist(name string) ([]string, error) {
	f, err := wordlists.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	words := []string{}
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}
	return words, scanner.Err()
}
