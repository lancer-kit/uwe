package main

import (
	"fmt"
	"time"

	"github.com/lancer-kit/uwe/v3"
	"github.com/lancer-kit/uwe/v3/presets"
)

func main() {
	job := presets.NewJob(3*time.Second, run)
	chief := uwe.NewChief()
	chief.AddWorker("cron-job", job)
	chief.Run()
}

func run() error {
	fmt.Println("i'm still alive")
	return nil
}
