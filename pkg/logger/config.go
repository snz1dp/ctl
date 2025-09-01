package logger

import (
	"encoding/json"
	// "os"
	// "path"
)

//Cfg 二次开发logger
func Cfg(logfile string) {

	// os.MkdirAll(path.Dir(logfile), os.ModePerm)

	config := logConfig{
		TimeFormat: "2006-01-02T15:04:05.000000Z",
		File: &fileLogger{
			Filename:   logfile,
			MaxSize:    1000,
			Daily:      false,
			Append:     true,
			PermitMask: "0644",
			LogLevel:   LevelDebug,
		},
	}
	cfg, _ := json.Marshal(config)
	SetLogger(string(cfg))
	SetLogPath(true)
	GetlocalLogger().DelLogger(AdapterConsole)
}

// Disable -
func Disable() {
	GetlocalLogger().DelLogger(AdapterConsole)
}
