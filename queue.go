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

	bulkPushSize = 100000
)

type LedgerRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

type BulkLedgerRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Bodies  [][]byte
}

// ItemBuilder creates a new item and returns a pointer to it.
// This is used when we load a segment of the queue from disk.
func LedgerRequestBuilder() interface{} {
	return &LedgerRequest{}
}

func newLedgerPushQueue() (*dque.DQue, error) {
	return dque.NewOrOpen("ledger-push-queue", queueDir, segmentSize, LedgerRequestBuilder)
}

func dequeue(q *dque.DQue) (*LedgerRequest, error) {
	iface, err := q.Dequeue()
	if err != nil {
		return nil, err
	}
	item, ok := iface.(*LedgerRequest)
	if !ok {
		return nil, errors.New("Could not convert dequeued item to LedgerRequest")
	}
	return item, nil
}

func startQueue(q *dque.DQue) {
	// Properly close a queue
	defer q.Close()

	for {
		// Sleep first so we don't immediately push while things are starting
		time.Sleep(time.Duration(queuePushIntervalMinutes) * time.Minute)

		pushCount := 0

		queueEmpty := false

		// Loop through and process all queued items until empty
		for !queueEmpty {

			groupedBulks := make(map[string]*BulkLedgerRequest)

			for i := 0; i < bulkPushSize; i++ {
				item, err := dequeue(q)
				if err == dque.ErrEmpty {
					queueEmpty = true
					break
				} else if err != nil {
					// Break out of loop and push if there's a dequeueing error
					fmt.Printf("Error dequeing: %s\n", err.Error())
					break
				}

				if !(item.Method == "POST" && item.Path == "/instance/measurements") {
					// Only measurements are bulk-able
					if err := pushLedgerRequest(item); err != nil {
						fmt.Printf("Error (single) pushing: %s\n", err.Error())
					}
					pushCount++

				} else {
					groupKey := item.Method + item.Path
					if _, exists := groupedBulks[groupKey]; !exists {
						groupedBulks[groupKey] = &BulkLedgerRequest{
							Method:  item.Method,
							Path:    item.Path,
							Headers: item.Headers,
							Bodies:  [][]byte{},
						}
					}
					groupedBulks[groupKey].Bodies = append(groupedBulks[groupKey].Bodies, item.Body)
				}
			}

			for _, bulk := range groupedBulks {
				bulkStr := ""
				for _, bodyBytes := range bulk.Bodies {
					bulkStr += string(bodyBytes)
				}

				lr := &LedgerRequest{
					Method:  bulk.Method,
					Path:    "/instance/multimeasurements",
					Headers: bulk.Headers,
					Body:    []byte(bulkStr),
				}

				// fmt.Println(bulkStr)

				if err := pushLedgerRequest(lr); err != nil {
					fmt.Printf("Error (bulk) pushing: %s\n", err.Error())
					continue
				}

				pushCount += len(bulk.Bodies)
			}
		}

		fmt.Printf("Dequeued and pushed %d items\n", pushCount)
	}
}
