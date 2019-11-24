package main

import (
	"fmt"
	"time"

	"github.com/lancer-kit/uwe/v2"
	"github.com/lancer-kit/uwe/v2/presets/cron"
)

func main() {
	job := cron.NewJob(3*time.Second, run)
	chief := uwe.NewChief()
	chief.AddWorker("cron-job", job)
	chief.Run()
}

func run() error {
	fmt.Println("i'm still alive")
	return nil
}