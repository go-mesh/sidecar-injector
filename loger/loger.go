package loger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ServiceComb/go-chassis/core/lager"
	log "github.com/Sirupsen/logrus"
)

var (
	once   sync.Once
	logDir string
)

// constant for loger file
const (
	PaasLager = "lager.log"
)

// constant values for logrotate parameters
const (
	LogRotateSize  = 10
	LogBackupCount = 7
)

// Initialize function will initialize the log file and start the log rotation
func Initialize() {
	fileName := filepath.Join(GetLogDir(), PaasLager)

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	Formatter := new(log.JSONFormatter)
	log.SetFormatter(Formatter)

	if err != nil {
		// Cannot open log file. Logging to stderr
		fmt.Println(err)
	} else {
		log.SetOutput(f)
	}

	initLogRotate(fileName)
}

// initLogRotate initialize log rotate
func initLogRotate(logFilePath string) {
	go func() {
		for {
			lager.LogRotate(filepath.Dir(logFilePath), LogRotateSize, LogBackupCount)
			time.Sleep(30 * time.Second)
		}
	}()
}

//GetLogDir is a function used to get the logging directory
func GetLogDir() string {
	once.Do(initDir)
	return logDir
}

func initDir() {
	wd, err := GetWorkDir()
	if err != nil {
		panic(err)
	}

	logDir = filepath.Join(wd, "log")
}

//GetWorkDir is a function used to get the working directory
func GetWorkDir() (string, error) {
	wd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return wd, nil
}
