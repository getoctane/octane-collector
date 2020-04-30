package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/joncrlsn/dque"
)

const (
	segmentSize = 50
)

type LedgerRequest struct {
	Method  string
	Path    string
	Body    []byte
	Headers http.Header
}

// ItemBuilder creates a new item and returns a pointer to it.
// This is used when we load a segment of the queue from disk.
func LedgerRequestBuilder() interface{} {
	return &LedgerRequest{}
}

func newLedgerPushQueue() (*dque.DQue, error) {
	return dque.NewOrOpen("ledger-push-queue", queueDir, segmentSize, LedgerRequestBuilder)
}

func dequeueAndPush(q *dque.DQue) error {
	iface, err := q.Dequeue()
	if err != nil {
		return err
	}

	item, ok := iface.(*LedgerRequest)
	if !ok {
		return errors.New("Could not convert dequeued item to LedgerRequest")
	}

	return pushLedgerRequest(item)
}

func startQueue(q *dque.DQue) {
	// Properly close a queue
	defer q.Close()

	for {
		// Sleep first so we don't immediately push while things are starting
		time.Sleep(time.Duration(queuePushIntervalMinutes) * time.Minute)

		pushCount := 0

		// Loop through and process all queued items until empty
		for {
			err := dequeueAndPush(q)
			if err == nil {
				pushCount++
				continue
			}
			if err == dque.ErrEmpty {
				break
			}
			fmt.Printf("Error dequeing and pushing: %s", err.Error())
			break
		}

		fmt.Printf("Dequeued and pushed %d items\n", pushCount)
	}
}
