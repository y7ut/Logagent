package agent

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/y7ut/logagent/pkg/file"
)

var (
	app *App
)

func Init(dataPath string, logPath string, logFile string) {

	err := file.PathExistOrCreate(dataPath)
	if err != nil {
		log.Fatalf("create runtime dir Error: %v", err)
	}

	err = file.PathExistOrCreate(logPath)
	if err != nil {
		log.Fatalf("create log dir Error: %v", err)
	}

	logFile = filepath.Join(logPath, logFile)
	writerLog, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModeAppend|os.ModePerm)
	if err != nil {
		log.Fatalf("create log File Error: %v", err)
	}
	log.Default().SetFlags(log.LstdFlags)
	log.Default().SetOutput(io.MultiWriter(writerLog, os.Stderr))

	app = NewApp(dataPath)
}

func Start() {
	app.Run()
}
