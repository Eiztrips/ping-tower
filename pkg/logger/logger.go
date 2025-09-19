package logger

import (
    "log"
    "os"
)

var (
    Logger *log.Logger
)

func init() {
    file, err := os.OpenFile("site_monitor.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatal(err)
    }

    Logger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func Info(message string) {
    Logger.Println("INFO: " + message)
}

func Error(message string) {
    Logger.Println("ERROR: " + message)
}