package game

import (
	"fmt"
	"sort"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// Game represents each game that multiple Players can participate in
	Game interface { // TODO: make struct
		Add(p player)
		Has(u db.User) bool
		Remove(u db.User)
		Start()
		Snag(p player)
		Swap(p player)
		Finish(p player)
	}
)

// TODO

func init() {
	// TODO: Use wordlist in game.  This is all just test code.
	var ws WordsSupplier = fileSystemWordsSupplier("/usr/share/dict/american-english")
	words, err := ws.Words()
	if err != nil {
		panic(err)
	}
	fmt.Println("there are", len(words), "words")
	s := make([]string, len(words))
	i := 0
	for w := range words {
		s[i] = w
		i++
	}
	sort.Strings(s)
	// fmt.Printf("sorted words are %v", s)
}
