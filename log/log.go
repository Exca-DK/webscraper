package log

import (
	"errors"
	"io"
	"log"
	"sync/atomic"
)

type Logger interface {
	Debug(...any)
	Info(...any)
	Warn(...any)
}

type LogLvl uint64

const (
	Debug LogLvl = 0
	Info  LogLvl = 1
	Warn  LogLvl = 2
)

func Lvls() []string {
	lvls := make([]string, 0)
	for i := Debug; i <= Warn; i++ {
		lvls = append(lvls, i.String())
	}
	return lvls
}

func (l LogLvl) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	}
	panic("unknown lvl")
}

func (l *LogLvl) FromString(str string) error {
	switch str {
	case "DEBUG":
		*l = Debug
		return nil
	case "INFO":
		*l = Info
		return nil
	case "WARN":
		*l = Warn
		return nil
	}
	return errors.New("unknown log lvl")
}

type logger struct {
	lvl atomic.Uint64

	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
}

func (l *logger) Debug(words ...any) {
	if !l.shouldLog(Debug) {
		return
	}
	l.log(l.debug, words...)
}

func (l *logger) Info(words ...any) {
	if !l.shouldLog(Info) {
		return
	}
	l.log(l.info, words...)
}

func (l *logger) Warn(words ...any) {
	if !l.shouldLog(Warn) {
		return
	}
	l.log(l.warn, words...)
}

func (l *logger) log(logger *log.Logger, words ...any) {
	logger.Println(words...)
}

func (l *logger) shouldLog(lvl LogLvl) bool {
	return LogLvl(l.lvl.Load()) <= lvl
}

func NewLogger(lvl LogLvl, output io.Writer) Logger {
	l := &logger{
		debug: log.New(output, Debug.String(), log.Ldate|log.Ltime|log.Lshortfile),
		info:  log.New(output, Info.String(), log.Ldate|log.Ltime|log.Lshortfile),
		warn:  log.New(output, Warn.String(), log.Ldate|log.Ltime|log.Lshortfile),
	}
	l.lvl.Store(uint64(lvl))
	return l
}
