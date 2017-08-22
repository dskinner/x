//go:generate go run gen.go

// Gopl is an REPL for text/template actions.
// TODO rename to Gpl.
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"
	"text/template"

	"dasa.cc/x/trigram"

	"github.com/chzyer/readline"
)

var root = template.New("gopl")

type auto struct{ trigram.Set }

func newauto() *auto {
	a := &auto{}
	for name := range pkgmap {
		a.Index(name)
	}
	return a
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// func (a auto) IsDynamic() bool { return true }

// https://github.com/chzyer/readline/blob/master/complete_segment.go#L63
// if position is zero, then i should save iter index and then add string to result
// ... and offset is for whatever last item is in newLine? or seems like pos should always equal line length
// and RetSegment from above makes it look like fuzzy match just doesn't work ..
//
// maybe sort so prefixed stuff shows first?
func (a auto) Do(line []rune, pos int) (newLine [][]rune, offset int) {
	ln := string(line[:pos])
	pos = 0
	p, n := a.Match(string(ln), 0.33)

	sort.Slice(p, func(i, j int) bool {
		return !strings.HasPrefix(p[j], ln) && strings.HasPrefix(p[i], ln)
	})

	for i, s := range p {
		t := strings.TrimPrefix(s, ln)
		d := len(s) - len(t)
		n[i] = float64(d)
		pos = max(pos, d)
		newLine = append(newLine, []rune(t+"."))
	}

	return newLine, pos
}

// TODO panic causes terminal issue
func main() {
	tmp, err := ioutil.TempFile("", "gopl")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmp.Name())

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "gopl: ",
		HistoryFile:       tmp.Name(),
		AutoComplete:      newauto(),
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	log.SetFlags(0)
	log.SetOutput(rl.Stderr())
	log.Println(runtime.Version())

	for i := 1; ; i++ {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)

		t := root.New(fmt.Sprintf("%v", i))
		if _, err := t.Parse(fmt.Sprintf("{{%s}}", line)); err != nil {
			log.Printf("%[1]T: %[1]v\n", err)
			continue
		}
		t.Execute(rl.Stderr(), pkgmap)
	}
}
