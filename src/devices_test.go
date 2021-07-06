package main

import (
	"context"
	"log"
	"testing"

	"gitlab.com/ms-ural/airport/core/logger.git"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGetdevicesFromSmartHome(t *testing.T) {

	var err error

	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Development: false,
	}

	zaplogger, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))

	if err != nil {
		log.Fatal(err)
	}
	defer zaplogger.Sync()

	msu = logger.NewMsuLogger(zaplogger, Product, Component)

	// if _, err = getUserDevicesFromSmartHome("http://192.168.10.17:9010"); err != nil {
	// 	msu.Error(context.TODO(), err)
	// }
	if _, err = getUserDevicesFromSmartHome(context.Background(), "11", "11", "http://188.226.37.223:9010"); err != nil {
		msu.Error(context.TODO(), err)
	}
}
