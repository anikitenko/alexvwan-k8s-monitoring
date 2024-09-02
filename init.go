package main

import (
	"github.com/BurntSushi/toml"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"path/filepath"
)

const (
	AppLogFile        = "app.log"
	RuntimeLogFile    = "runtime.log"
	ConfigFileName    = "config.toml"
	LogDirectoryName  = "log"
	DataDirectoryName = "data"

	GlobalCollection      = "global"
	DataCollection        = "data"
	SessionsCollection    = "sessions"
	KubeconfigsCollection = "kubeconfigs"
	ActivityConsole       = "activity_console"

	HttpSessionDurationSeconds = 432000
)

var (
	CurrentDirectory  string
	ConfigFile        string
	LogDirectory      string
	PathSeparator     = string(filepath.Separator)
	RuntimeLogger     *log.Logger
	DataDirectory     string
	DBHelper          *DatabaseHelper
	ConfigurationMode bool
)

func init() {
	var err error
	CurrentDirectory, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logger.Fatalf("Problem with finding application path: %v", err)
	}

	DataDirectory = CurrentDirectory + PathSeparator + DataDirectoryName
	ConfigFile = DataDirectory + PathSeparator + ConfigFileName
	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		logger.Fatalf("Config file is not exists at %s", ConfigFile)
	}

	var config Config
	if _, err := toml.DecodeFile(ConfigFile, &config); err != nil {
		logger.Fatalf("Failed to parse config file: %v", err)
	}

	LogDirectory = CurrentDirectory + PathSeparator + LogDirectoryName
	if _, err := os.Stat(LogDirectory); os.IsNotExist(err) {
		if err := os.Mkdir(LogDirectory, 0755); err != nil {
			logger.Fatalf("Problem with creating directory %s: %v", LogDirectory, err)
		}
	}

	if config.Log.MaxSize == 0 {
		config.Log.MaxSize = 1
	}
	if config.Log.MaxBackups == 0 {
		config.Log.MaxBackups = 5
	}
	if config.Log.MaxAge == 0 {
		config.Log.MaxAge = 7
	}
	logger.SetFormatter(&logger.JSONFormatter{})
	lumberjackLogger := &lumberjack.Logger{
		Filename:   LogDirectory + PathSeparator + AppLogFile,
		MaxSize:    config.Log.MaxSize,
		MaxBackups: config.Log.MaxBackups,
		MaxAge:     config.Log.MaxAge,
		LocalTime:  config.Log.LocalTime,
		Compress:   config.Log.Compress,
	}
	lumberjackLoggerRuntime := &lumberjack.Logger{
		Filename:   LogDirectory + PathSeparator + RuntimeLogFile,
		MaxSize:    config.Log.MaxSize,
		MaxBackups: config.Log.MaxBackups,
		MaxAge:     config.Log.MaxAge,
		LocalTime:  config.Log.LocalTime,
		Compress:   config.Log.Compress,
	}
	logger.SetOutput(lumberjackLogger)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(lumberjackLoggerRuntime)
	RuntimeLogger = log.New(lumberjackLoggerRuntime, "", log.LstdFlags|log.Lshortfile)
	logger.Info("Starting... Application logging was configured successfully")

	// Make sure data directory exists
	if _, err := os.Stat(DataDirectory); os.IsNotExist(err) {
		if err := os.Mkdir(DataDirectory, 0755); err != nil {
			logger.Fatalf("Problem with creating directory %s: %v", DataDirectory, err)
		}
	}
	logger.Info("Data directory check completed")
}
