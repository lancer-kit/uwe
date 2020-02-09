package socket

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_socketServer_Serve(t *testing.T) {
	socketName := "/tmp/uwe_test.socket"
	sw := NewServer(socketName, Action{
		Name: "ping",
		Handler: func(_ Request) Response {
			return Response{Status: StatusOk, Data: "pong"}
		},
	})
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		if err := sw.Serve(ctx); err != nil {
			log.Print("err:", err)
		}

		done <- struct{}{}
	}()

	go func() {
		for {
			select {
			case err := <-sw.Errors():
				log.Print(err)
			case <-ctx.Done():
				done <- struct{}{}
				return
			}
		}
	}()

	time.Sleep(time.Second)

	client := NewClient(socketName)
	req := Request{Action: "ping"}
	log.Println("Send ping")

	resp, err := client.Send(req)
	log.Println("Got Response:", resp)
	assert.NoError(t, err)

	assert.Equal(t, StatusOk, resp.Status)
	assert.Equal(t, "pong", resp.Data)

	cancel()

	<-done
	<-done

}
