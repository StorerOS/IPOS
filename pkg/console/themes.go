package console

import "github.com/fatih/color"

var (
	Theme = map[string]*color.Color{
		"Debug":  color.New(color.FgWhite, color.Faint, color.Italic),
		"Fatal":  color.New(color.FgRed, color.Italic, color.Bold),
		"Error":  color.New(color.FgYellow, color.Italic),
		"Info":   color.New(color.FgGreen, color.Bold),
		"Print":  color.New(),
		"PrintB": color.New(color.FgBlue, color.Bold),
		"PrintC": color.New(color.FgGreen, color.Bold),
	}
)

func SetColorOff() {
	privateMutex.Lock()
	defer privateMutex.Unlock()
	color.NoColor = true
}

func SetColorOn() {
	privateMutex.Lock()
	defer privateMutex.Unlock()
	color.NoColor = false
}

func SetColor(tag string, cl *color.Color) {
	privateMutex.Lock()
	defer privateMutex.Unlock()
	Theme[tag] = cl
}
