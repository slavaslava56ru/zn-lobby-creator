package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"zn/hub"
	"zn/websocket"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var port = flag.Int("port", 8080, "http service address")

func main() {
	flag.Parse()

	consoleDebugging := zapcore.Lock(os.Stdout)
	logger := zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), consoleDebugging, zap.NewAtomicLevelAt(zapcore.DebugLevel)), zap.AddCaller()).Sugar()
	defer logger.Sync()

	hub := hub.NewHub(logger)
	go hub.Run()

	wsServer := websocket.NewServer(hub, *port, logger)
	wsServer.Start()
	defer wsServer.Close()

	logger.Info("App started")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	<-ch
	signal.Stop(ch)
	logger.Info("App stopped")
}
