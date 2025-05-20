package logger

import (
	"io"
	"log"
	"os"
)

var (
	LogFile *os.File
	Logger  *log.Logger
)

func Init(logFilePath string) error {
	// Create or append to the log file
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	LogFile = file

	// Create a multi-writer to write to stdout and file
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Initialize the logger
	Logger = log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	return nil
}

func Close() {
	if LogFile != nil {
		LogFile.Close()
	}
}
