package logging

import (
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/finnan444/utils/time/cron"
)

// LoggersConsts
const (
	ErrorLogger = "errorLogger"
	StdLogger   = "logger"
)

var (
	logLocker     sync.RWMutex
	files         = make(map[string]*rotater)
	customLoggers = make(map[string]*log.Logger)
)

type rotater struct {
	sync.RWMutex
	filename string
	file     *os.File
}

// Write satisfies the io.Writer interface.
func (w *rotater) Write(output []byte) (int, error) {
	w.RLock()
	defer w.RUnlock()
	return w.file.Write(output)
}

// Perform the actual act of rotating and reopening file.
func (w *rotater) Rotate() (err error) {
	w.Lock()
	defer w.Unlock()

	// Close existing file if open
	if w.file != nil {
		err = w.file.Close()
		w.file = nil
		if err != nil {
			return
		}
	}

	// Rename dest file if it already exists
	_, err = os.Stat(w.filename)
	if err == nil {
		err = os.Rename(w.filename, w.filename+"."+time.Now().UTC().Format("2006-01-02-15-04"))
		if err != nil {
			return
		}
	}

	// Create a file.
	w.file, err = os.Create(w.filename)
	return
}

func (w *rotater) rotate() {
	_ = w.Rotate()
}

func newRotater(filename string, period time.Duration) *rotater {
	result := &rotater{filename: filename}
	if err := result.Rotate(); err != nil {
		return nil
	}
	cron.Add(period, time.UTC, result.rotate)
	return result
}

// GetErrorLogger returns files logger with stderr
func GetErrorLogger(name string, restartPeriod time.Duration, filenames ...string) (result *log.Logger) {
	var ok bool
	logLocker.RLock()
	if result, ok = customLoggers[name]; ok {
		logLocker.RUnlock()
		return
	}
	logLocker.RUnlock()
	var writers []io.Writer
	writers = append(writers, os.Stderr)
	var file *rotater
	logLocker.Lock()
	for _, fn := range filenames {
		if file, ok = files[fn]; !ok {
			if file = newRotater(fn, restartPeriod); file != nil {
				files[fn] = file
			}
		}
		if file != nil {
			writers = append(writers, file)
		}
	}
	result = log.New(io.MultiWriter(writers...), "", log.LstdFlags)
	customLoggers[name] = result
	logLocker.Unlock()
	return
}

// GetLogger returns logger with stdout
func GetLogger(name string, restartPeriod time.Duration, filenames ...string) (result *log.Logger) {
	var ok bool
	logLocker.RLock()
	if result, ok = customLoggers[name]; ok {
		logLocker.RUnlock()
		return
	}
	logLocker.RUnlock()
	writers := make([]io.Writer, 0, len(filenames)+1)
	writers = append(writers, os.Stdout)
	var file *rotater
	logLocker.Lock()
	for _, fn := range filenames {
		if file, ok = files[fn]; !ok {
			file = newRotater(fn, restartPeriod)
			files[fn] = file
		}
		writers = append(writers, file)
	}
	result = log.New(io.MultiWriter(writers...), "", log.LstdFlags)
	customLoggers[name] = result
	logLocker.Unlock()
	return
}

//LogrusInit универсальный инициатор лога ошибок
func LogrusInit(path string) (erL *logrus.Logger) {
	erL = logrus.New()
	erL.SetOutput(os.Stderr)
	erL.SetFormatter(&logrus.JSONFormatter{})
	erL.SetReportCaller(true)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		erL.Out = file
	} else {
		erL.Info("Failed to log to file, using default stderr")
	}
	return
}
