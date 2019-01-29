package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync/atomic"
)

const (
	LevelDebug int32 = 1 << iota
	LevelInfo
	LevelWarn
	LevelError
)

const MaxBytes int = 10 * 1024 * 1024
const BackupCount int = 10

type Logger struct {
	Level int32

	Debug *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
	Fatal *log.Logger

	LogFile *RotatingFileHandler
}

var defaultLogger Logger

// DefaultFlags used by created loggers
var DefaultFlags = log.Ldate | log.Ltime | log.Lshortfile

// RotatingFileHandler writes log a file, if file size exceeds maxBytes,
// it will backup current file and open a new one.
//
// max backup file number is set by backupCount, it will delete oldest if backups too many.
type RotatingFileHandler struct {
	fd          *os.File
	fileName    string
	maxBytes    int
	backupCount int
}

// NewRotatingFileHandler creates dirs and opens the logfile
func NewRotatingFileHandler(fileName string, maxBytes int, backupCount int) (*RotatingFileHandler, error) {
	dir := path.Dir(fileName)
	var err error
	err = os.Mkdir(dir, 0775)
	if err != nil && err != os.ErrExist {
		return nil, err
	}

	h := new(RotatingFileHandler)
	if maxBytes <= 0 {
		return nil, fmt.Errorf("invalid max bytes")
	}
	h.fileName = fileName
	h.maxBytes = maxBytes
	h.backupCount = backupCount

	h.fd, err = os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (h *RotatingFileHandler) Write(p []byte) (n int, err error) {
	h.doRollover()
	return h.fd.Write(p)
}

func (h *RotatingFileHandler) Close() error {
	if h.fd != nil {
		return h.fd.Close()
	}
	return nil
}

func (h *RotatingFileHandler) doRollover() {
	f, err := h.fd.Stat()
	if err != nil {
		return
	}

	if h.maxBytes <= 0 {
		return
	} else if f.Size() < int64(h.maxBytes) {
		return
	}

	if h.backupCount > 0 {
		h.fd.Close()

		for i := h.backupCount - 1; i > 0; i-- {
			sfn := fmt.Sprintf("%s.%d", h.fileName, i)
			dfn := fmt.Sprintf("%s.%d", h.fileName, i+1)
			os.Rename(sfn, dfn)
		}

		dfn := fmt.Sprintf("%s.1", h.fileName)
		os.Rename(h.fileName, dfn)
		h.fd, _ = os.OpenFile(h.fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}
}

func Start(level int32, path string) {
	doLogging(level, path, MaxBytes, BackupCount)
}

func StartEx(level int32, path string, maxBytes, backupCount int) {
	doLogging(level, path, maxBytes, backupCount)
}

func Stop() error {
	if defaultLogger.LogFile != nil {
		return defaultLogger.LogFile.Close()
	}
	return nil
}

//Sync commits the current contents of the file to stable storage.
//Typically, this means flushing the file system's in-memory copy
//of recently written data to disk.
func Sync() {
	if defaultLogger.LogFile != nil {
		defaultLogger.LogFile.fd.Sync()
	}
}

func doLogging(logLevel int32, fileName string, maxBytes, backupCount int) {
	debugHandle := ioutil.Discard
	infoHandle := ioutil.Discard
	warnHandle := ioutil.Discard
	errorHandle := ioutil.Discard
	fatalHandle := ioutil.Discard

	var fileHandler *RotatingFileHandler

	switch logLevel {
	case LevelDebug:
		debugHandle = os.Stdout
		fallthrough
	case LevelInfo:
		infoHandle = os.Stdout
		fallthrough
	case LevelWarn:
		warnHandle = os.Stdout
		fallthrough
	case LevelError:
		errorHandle = os.Stderr
		fatalHandle = os.Stderr
	}

	if fileName != "" {
		var err error
		fileHandler, err = NewRotatingFileHandler(fileName, maxBytes, backupCount)
		if err != nil {
			log.Fatal("unable to create RotatingFileHandler: ", err)
		}

		if debugHandle == os.Stdout {
			debugHandle = io.MultiWriter(fileHandler, debugHandle)
		}

		if infoHandle == os.Stdout {
			infoHandle = io.MultiWriter(fileHandler, infoHandle)
		}

		if warnHandle == os.Stdout {
			warnHandle = io.MultiWriter(fileHandler, warnHandle)
		}

		if errorHandle == os.Stderr {
			errorHandle = io.MultiWriter(fileHandler, errorHandle)
		}

		if fatalHandle == os.Stderr {
			fatalHandle = io.MultiWriter(fileHandler, fatalHandle)
		}
	}

	defaultLogger = Logger{
		Debug:   log.New(debugHandle, "DEBUG: ", DefaultFlags),
		Info:    log.New(infoHandle, "INFO: ", DefaultFlags),
		Warn:    log.New(warnHandle, "WARNING: ", DefaultFlags),
		Error:   log.New(errorHandle, "ERROR: ", DefaultFlags),
		Fatal:   log.New(errorHandle, "FATAL: ", DefaultFlags),
		LogFile: fileHandler,
	}

	atomic.StoreInt32(&defaultLogger.Level, int32(logLevel))
}

func Debug(format string, a ...interface{}) {
	defaultLogger.Debug.Output(2, fmt.Sprintf(format, a...))
}

func Info(format string, a ...interface{}) {
	defaultLogger.Info.Output(2, fmt.Sprintf(format, a...))
}

func Warn(format string, a ...interface{}) {
	defaultLogger.Warn.Output(2, fmt.Sprintf(format, a...))
}

func Error(err error) {
	defaultLogger.Error.Output(2, fmt.Sprintf("%s\n", err))
}

func Errorf(format string, a ...interface{}) {
	defaultLogger.Error.Output(2, fmt.Sprintf(format, a))
}

// Fatal writes to the Fatal destination and exits with an error 255 code
func Fatal(a ...interface{}) {
	defaultLogger.Fatal.Output(2, fmt.Sprint(a...))
	Sync()
	os.Exit(255)
}

// Fatalf writes to the Fatal destination and exits with an error 255 code
func Fatalf(format string, a ...interface{}) {
	defaultLogger.Fatal.Output(2, fmt.Sprintf(format, a...))
	Sync()
	os.Exit(255)
}

