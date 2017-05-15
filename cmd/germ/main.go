// Command germ haphazardly runs interactive shells; use may cause illness.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"unsafe"

	"github.com/google/gxui"
	"github.com/google/gxui/drivers/gl"
	"github.com/google/gxui/gxfont"
	"github.com/google/gxui/math"
	"github.com/google/gxui/mixins"
	"github.com/google/gxui/themes/dark"
)

var (
	flagExec    = flag.String("exec", "bash", "Command to execute.")
	flagScaling = flag.Float64("scaling", 1.0, "Adjusts the scaling of UI rendering.")
)

type Theme struct {
	*dark.Theme
}

func CreateTheme(driver gxui.Driver) *Theme {
	t := &Theme{}
	t.Theme = dark.CreateTheme(driver).(*dark.Theme)
	font, err := driver.CreateFont(gxfont.Monospace, 15)
	if err != nil {
		log.Fatalf("Failed to load monospace font - %v\n", err)
	}
	font.LoadGlyphs(32, 126)
	t.SetDefaultFont(font)

	return t
}

type Term struct {
	mixins.CodeEditor

	driver gxui.Driver
	theme  *Theme
	ctrl   *gxui.TextBoxController

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func CreateTerm(theme *Theme) *Term {
	t := &Term{}
	t.driver = theme.Driver()
	t.theme = theme
	t.Init(t, t.driver, theme, theme.DefaultFont())
	t.SetTextColor(theme.TextBoxDefaultStyle.FontColor)
	t.SetMargin(math.Spacing{L: 3, T: 3, R: 3, B: 3})
	t.SetPadding(math.Spacing{L: 3, T: 3, R: 3, B: 3})
	t.SetBorderPen(gxui.TransparentPen)
	t.SetScrollBarEnabled(true)

	t.OnKeyUp(func(ev gxui.KeyboardEvent) {
		if ev.Key == gxui.KeyEnter {
			for _, caret := range t.Carets() {
				idx := t.LineIndex(caret) - 1
				s := t.LineStart(idx)
				e := t.LineEnd(idx)
				line := t.TextAt(s, e) + "\n"
				t.stdin.Write([]byte(line))
				t.ScrollToLine(t.List.Adapter().Count())
			}
		}
	})

	// TODO workout new exported methods on gxui for missing bits I want
	elem := reflect.ValueOf(&t.TextBox).Elem()
	field := elem.FieldByName("controller")
	addr := field.Pointer()
	t.ctrl = (*gxui.TextBoxController)(unsafe.Pointer(addr))

	return t
}

func (t *Term) CreateSuggestionList() gxui.List {
	l := t.theme.CreateList()
	l.SetBackgroundBrush(t.theme.CodeSuggestionListStyle.Brush)
	l.SetBorderPen(t.theme.CodeSuggestionListStyle.Pen)
	return l
}

func (t *Term) Exec(name string, arg ...string) (err error) {
	if t.cmd != nil {
		return fmt.Errorf("Attempting to exec but a process is already running.")
	}

	t.cmd = exec.Command(name, arg...)

	if t.stdin, err = t.cmd.StdinPipe(); err != nil {
		return fmt.Errorf("Failed to create stdin pipe - %v\n", err)
	}

	if t.stdout, err = t.cmd.StdoutPipe(); err != nil {
		return fmt.Errorf("Failed to create stdout pipe - %v\n", err)
	}

	if t.stderr, err = t.cmd.StderrPipe(); err != nil {
		return fmt.Errorf("Failed to create stderr pipe - %v\n", err)
	}

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("Failed to start process - %v\n", err)
	}

	t.CombinedEcho()

	// makes t.cmd.ProcessState available after process exits
	go t.cmd.Wait()

	return nil
}

func (t *Term) Kill() error {
	if err := t.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("Failed to kill process - %v\n", err)
	}
	return nil
}

func (t *Term) Echo() {
	go t.echo(t.stdout)
}

func (t *Term) CombinedEcho() {
	go t.echo(t.stdout)
	go t.echo(t.stderr)
}

func (t *Term) echo(reader io.Reader) {
	r := bufio.NewReader(reader)
	for t.cmd.ProcessState == nil {
		line, err := r.ReadBytes('\n')

		switch {
		case err == io.EOF && len(line) == 0:
			return
		case err == io.EOF:
			log.Println("Unexpected EOF while echoing process.")
		case err != nil:
			log.Fatalf("Echoing process failed - %v\n", err)
		}

		t.driver.CallSync(func() {
			t.ctrl.ReplaceAllRunes(bytes.Runes(line))
			t.ctrl.Deselect(false)
			t.ScrollToLine(t.List.Adapter().Count())
		})
	}
}

func appMain(driver gxui.Driver) {
	theme := CreateTheme(driver)
	window := theme.CreateWindow(800, 480, "germ")
	window.SetScale(float32(*flagScaling))

	term := CreateTerm(theme)
	if err := term.Exec(*flagExec); err != nil {
		log.Fatal(err)
	}

	window.AddChild(term)
	window.SetFocus(term)

	window.OnResize(func() {
		size := window.Viewport().SizePixels()
		term.SetDesiredWidth(size.W)
	})

	window.OnMouseUp(func(ev gxui.MouseEvent) {
		window.SetFocus(term)
	})

	window.OnClose(func() {
		term.Kill()
		driver.Terminate()
	})
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nHaphazardly runs interactive shells. Use may cause illness.\n")
}

func main() {
	flag.Usage = usage
	flag.Parse()
	gl.StartDriver(appMain)
}
