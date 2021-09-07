package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/storeros/ipos/cmd/ipos/logger/message/log"
	"github.com/storeros/ipos/pkg/color"
	c "github.com/storeros/ipos/pkg/console"
)

type Logger interface {
	json(msg string, args ...interface{})
	quiet(msg string, args ...interface{})
	pretty(msg string, args ...interface{})
}

func consoleLog(console Logger, msg string, args ...interface{}) {
	switch {
	case jsonFlag:
		msg = ansiRE.ReplaceAllLiteralString(msg, "")
		console.json(msg, args...)
	case quietFlag:
		console.quiet(msg, args...)
	default:
		console.pretty(msg, args...)
	}
}

func Fatal(err error, msg string, data ...interface{}) {
	fatal(err, msg, data...)
}

func fatal(err error, msg string, data ...interface{}) {
	var errMsg string
	if msg != "" {
		errMsg = errorFmtFunc(fmt.Sprintf(msg, data...), err, jsonFlag)
	} else {
		errMsg = err.Error()
	}
	consoleLog(fatalMessage, errMsg)
}

var fatalMessage fatalMsg

type fatalMsg struct {
}

func (f fatalMsg) json(msg string, args ...interface{}) {
	logJSON, err := json.Marshal(&log.Entry{
		Level: FatalLvl.String(),
		Time:  time.Now().UTC().Format(time.RFC3339Nano),
		Trace: &log.Trace{Message: fmt.Sprintf(msg, args...), Source: []string{getSource(6)}},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(logJSON))

	os.Exit(1)

}

func (f fatalMsg) quiet(msg string, args ...interface{}) {
	f.pretty(msg, args...)
}

var (
	logTag      = "ERROR"
	logBanner   = color.BgRed(color.FgWhite(color.Bold(logTag))) + " "
	emptyBanner = color.BgRed(strings.Repeat(" ", len(logTag))) + " "
	bannerWidth = len(logTag) + 1
)

func (f fatalMsg) pretty(msg string, args ...interface{}) {
	errMsg := fmt.Sprintf(msg, args...)

	tagPrinted := false

	for _, line := range strings.Split(errMsg, "\n") {
		if len(line) == 0 {
			break
		}

		for {
			ansiSaveAttributes()
			if !tagPrinted {
				c.Print(logBanner)
				tagPrinted = true
			} else {
				c.Print(emptyBanner)
			}
			ansiRestoreAttributes()
			ansiMoveRight(bannerWidth)
			c.Println(line)
			break
		}
	}

	os.Exit(1)
}

type infoMsg struct{}

var info infoMsg

func (i infoMsg) json(msg string, args ...interface{}) {
	logJSON, err := json.Marshal(&log.Entry{
		Level:   InformationLvl.String(),
		Message: fmt.Sprintf(msg, args...),
		Time:    time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(logJSON))
}

func (i infoMsg) quiet(msg string, args ...interface{}) {
	i.pretty(msg, args...)
}

func (i infoMsg) pretty(msg string, args ...interface{}) {
	c.Printf(msg, args...)
}

func Info(msg string, data ...interface{}) {
	consoleLog(info, msg+"\n", data...)
}

var startupMessage startUpMsg

type startUpMsg struct {
}

func (s startUpMsg) json(msg string, args ...interface{}) {
}

func (s startUpMsg) quiet(msg string, args ...interface{}) {
}

func (s startUpMsg) pretty(msg string, args ...interface{}) {
	c.Printf(msg, args...)
}

func StartupMessage(msg string, data ...interface{}) {
	consoleLog(startupMessage, msg+"\n", data...)
}
