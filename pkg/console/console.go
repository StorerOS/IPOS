package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

var (
	DebugPrint = false

	publicMutex  = &sync.Mutex{}
	privateMutex = &sync.Mutex{}

	stderrColoredOutput = colorable.NewColorableStderr()

	Print = func(data ...interface{}) {
		consolePrint("Print", Theme["Print"], data...)
	}

	PrintC = func(data ...interface{}) {
		consolePrint("PrintC", Theme["PrintC"], data...)
	}

	Printf = func(format string, data ...interface{}) {
		consolePrintf("Print", Theme["Print"], format, data...)
	}

	Println = func(data ...interface{}) {
		consolePrintln("Print", Theme["Print"], data...)
	}

	Fatal = func(data ...interface{}) {
		consolePrint("Fatal", Theme["Fatal"], data...)
		os.Exit(1)
	}

	Fatalf = func(format string, data ...interface{}) {
		consolePrintf("Fatal", Theme["Fatal"], format, data...)
		os.Exit(1)
	}

	Fatalln = func(data ...interface{}) {
		consolePrintln("Fatal", Theme["Fatal"], data...)
		os.Exit(1)
	}

	Error = func(data ...interface{}) {
		consolePrint("Error", Theme["Error"], data...)
	}

	Errorf = func(format string, data ...interface{}) {
		consolePrintf("Error", Theme["Error"], format, data...)
	}

	Errorln = func(data ...interface{}) {
		consolePrintln("Error", Theme["Error"], data...)
	}

	Info = func(data ...interface{}) {
		consolePrint("Info", Theme["Info"], data...)
	}

	Infof = func(format string, data ...interface{}) {
		consolePrintf("Info", Theme["Info"], format, data...)
	}

	Infoln = func(data ...interface{}) {
		consolePrintln("Info", Theme["Info"], data...)
	}

	Debug = func(data ...interface{}) {
		if DebugPrint {
			consolePrint("Debug", Theme["Debug"], data...)
		}
	}

	Debugf = func(format string, data ...interface{}) {
		if DebugPrint {
			consolePrintf("Debug", Theme["Debug"], format, data...)
		}
	}

	Debugln = func(data ...interface{}) {
		if DebugPrint {
			consolePrintln("Debug", Theme["Debug"], data...)
		}
	}

	Colorize = func(tag string, data interface{}) string {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			colorized, ok := Theme[tag]
			if ok {
				return colorized.SprintFunc()(data)
			}
		}
		return fmt.Sprint(data)
	}

	Eraseline = func() {
		consolePrintf("Print", Theme["Print"], "%c[2K\n", 27)
		consolePrintf("Print", Theme["Print"], "%c[A", 27)
	}
)

func consolePrint(tag string, c *color.Color, a ...interface{}) {
	privateMutex.Lock()
	defer privateMutex.Unlock()

	switch tag {
	case "Debug":
		if len(a) == 0 {
			return
		}
		output := color.Output
		color.Output = stderrColoredOutput
		if isatty.IsTerminal(os.Stderr.Fd()) {
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Print(a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": <DEBUG> ")
			fmt.Fprint(color.Output, a...)
		}
		color.Output = output
	case "Fatal":
		fallthrough
	case "Error":
		if len(a) == 0 {
			return
		}
		output := color.Output
		color.Output = stderrColoredOutput
		if isatty.IsTerminal(os.Stderr.Fd()) {
			c.Print(ProgramName() + ": <ERROR> ")
			c.Print(a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": <ERROR> ")
			fmt.Fprint(color.Output, a...)
		}
		color.Output = output
	case "Info":
		if len(a) == 0 {
			return
		}
		if isatty.IsTerminal(os.Stdout.Fd()) {
			c.Print(ProgramName() + ": ")
			c.Print(a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": ")
			fmt.Fprint(color.Output, a...)
		}
	default:
		if isatty.IsTerminal(os.Stdout.Fd()) {
			c.Print(a...)
		} else {
			fmt.Fprint(color.Output, a...)
		}
	}
}

func consolePrintf(tag string, c *color.Color, format string, a ...interface{}) {
	privateMutex.Lock()
	defer privateMutex.Unlock()

	switch tag {
	case "Debug":
		if len(a) == 0 {
			return
		}
		output := color.Output
		color.Output = stderrColoredOutput
		if isatty.IsTerminal(os.Stderr.Fd()) {
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Printf(format, a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": <DEBUG> ")
			fmt.Fprintf(color.Output, format, a...)
		}
		color.Output = output
	case "Fatal":
		fallthrough
	case "Error":
		if len(a) == 0 {
			return
		}
		output := color.Output
		color.Output = stderrColoredOutput
		if isatty.IsTerminal(os.Stderr.Fd()) {
			c.Print(ProgramName() + ": <ERROR> ")
			c.Printf(format, a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": <ERROR> ")
			fmt.Fprintf(color.Output, format, a...)
		}
		color.Output = output
	case "Info":
		if len(a) == 0 {
			return
		}
		if isatty.IsTerminal(os.Stdout.Fd()) {
			c.Print(ProgramName() + ": ")
			c.Printf(format, a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": ")
			fmt.Fprintf(color.Output, format, a...)
		}
	default:
		if isatty.IsTerminal(os.Stdout.Fd()) {
			c.Printf(format, a...)
		} else {
			fmt.Fprintf(color.Output, format, a...)
		}
	}
}

func consolePrintln(tag string, c *color.Color, a ...interface{}) {
	privateMutex.Lock()
	defer privateMutex.Unlock()

	switch tag {
	case "Debug":
		if len(a) == 0 {
			return
		}
		output := color.Output
		color.Output = stderrColoredOutput
		if isatty.IsTerminal(os.Stderr.Fd()) {
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Println(a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": <DEBUG> ")
			fmt.Fprintln(color.Output, a...)
		}
		color.Output = output
	case "Fatal":
		fallthrough
	case "Error":
		if len(a) == 0 {
			return
		}
		output := color.Output
		color.Output = stderrColoredOutput
		if isatty.IsTerminal(os.Stderr.Fd()) {
			c.Print(ProgramName() + ": <ERROR> ")
			c.Println(a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": <ERROR> ")
			fmt.Fprintln(color.Output, a...)
		}
		color.Output = output
	case "Info":
		if len(a) == 0 {
			return
		}
		if isatty.IsTerminal(os.Stdout.Fd()) {
			c.Print(ProgramName() + ": ")
			c.Println(a...)
		} else {
			fmt.Fprint(color.Output, ProgramName()+": ")
			fmt.Fprintln(color.Output, a...)
		}
	default:
		if isatty.IsTerminal(os.Stdout.Fd()) {
			c.Println(a...)
		} else {
			fmt.Fprintln(color.Output, a...)
		}
	}
}

func Lock() {
	publicMutex.Lock()
}

func Unlock() {
	publicMutex.Unlock()
}

func ProgramName() string {
	_, progName := filepath.Split(os.Args[0])
	return progName
}

type Table struct {
	RowColors []*color.Color

	AlignRight []bool

	TableIndentWidth int
}

func NewTable(rowColors []*color.Color, alignRight []bool, indentWidth int) *Table {
	return &Table{rowColors, alignRight, indentWidth}
}

func (t *Table) DisplayTable(rows [][]string) error {
	numRows := len(rows)
	numCols := len(rows[0])
	if numRows != len(t.RowColors) {
		return fmt.Errorf("row count and row-colors mismatch")
	}

	maxColWidths := make([]int, numCols)
	for _, row := range rows {
		if len(row) != len(t.AlignRight) {
			return fmt.Errorf("col count and align-right mismatch")
		}
		for i, v := range row {
			if len([]rune(v)) > maxColWidths[i] {
				maxColWidths[i] = len([]rune(v))
			}
		}
	}

	paddedText := make([][]string, numRows)
	for r, row := range rows {
		paddedText[r] = make([]string, numCols)
		for c, cell := range row {
			if t.AlignRight[c] {
				fmtStr := fmt.Sprintf("%%%ds", maxColWidths[c])
				paddedText[r][c] = fmt.Sprintf(fmtStr, cell)
			} else {
				extraWidth := maxColWidths[c] - len([]rune(cell))
				fmtStr := fmt.Sprintf("%%s%%%ds", extraWidth)
				paddedText[r][c] = fmt.Sprintf(fmtStr, cell, "")
			}
		}
	}

	segments := make([]string, numCols)
	for i, c := range maxColWidths {
		segments[i] = strings.Repeat("─", c+2)
	}
	indentText := strings.Repeat(" ", t.TableIndentWidth)
	border := fmt.Sprintf("%s┌%s┐", indentText, strings.Join(segments, "┬"))
	fmt.Println(border)

	for r, row := range paddedText {
		fmt.Print(indentText + "│ ")
		for c, text := range row {
			t.RowColors[r].Print(text)
			if c != numCols-1 {
				fmt.Print(" │ ")
			}
		}
		fmt.Println(" │")
	}

	border = fmt.Sprintf("%s└%s┘", indentText, strings.Join(segments, "┴"))
	fmt.Println(border)

	return nil
}

func RewindLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Printf("\033[1A\033[K")
	}
}
