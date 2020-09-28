package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/getoctane/octane-collector/util"
	"github.com/joncrlsn/dque"
)

const (
	segmentSize = 50

	bulkPushSize = 100
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
	q, err := dque.NewOrOpen("ledger-push-queue", queueDir, segmentSize, LedgerRequestBuilder)
	if err != nil {
		return nil, err
	}
	if err := q.TurboOn(); err != nil {
		return nil, err
	}
	return q, nil
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

type acknowledgement struct {
	Success bool `json:"success"`
}

func pushOrRequeue(q *dque.DQue, lr *LedgerRequest) error {
	if respBody, err := pushLedgerRequest(lr); err != nil {

		// TODO(ben): We should probably check for 502 or 503 here, just to make
		// sure we re-queue when we get (likely from a LoadBalancer) HTTP errors
		// that represent connection errors to the backend.
		if errHTTP, isHTTP := err.(*util.ErrorHTTP); isHTTP {
			fmt.Printf("Error HTTP pushing to ledger (won't re-queue): %s\n", errHTTP.Error())
			return errHTTP
		}

		fmt.Println(string(respBody))

		dst := []*acknowledgement{}
		if jsonErr := json.Unmarshal(respBody, &dst); jsonErr != nil {
			retErr := fmt.Errorf("Error unmarshalling JSON response from multi-measurements: %s", jsonErr.Error())
			fmt.Println(retErr)
			return retErr
		}
		allGood := true
		for _, ack := range dst {
			if !ack.Success {
				allGood = false
				break
			}
		}
		if !allGood {
			badErr := fmt.Errorf("Partial failure: %s\n", string(respBody))
			fmt.Println(badErr)
			return badErr
		}

		fmt.Printf("Error pushing to ledger: %s\n", err.Error())
		if err := q.Enqueue(lr); err != nil {
			fmt.Printf("Error re-queueing: %s\n", err.Error())
			return err
		}
	}
	return nil
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
					if err := pushOrRequeue(q, item); err != nil {
						continue
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
				bulkBytes := []byte{}
				for _, bodyBytes := range bulk.Bodies {
					bulkBytes = append(bulkBytes, bodyBytes...)
				}

				// fmt.Println(string(bulkBytes))

				fmt.Printf("Posting bulk with bytesize of %d\n", len(bulkBytes))

				lr := &LedgerRequest{
					Method:  bulk.Method,
					Path:    "/instance/multimeasurements/v2",
					Headers: bulk.Headers,
					Body:    bulkBytes,
				}

				if err := pushOrRequeue(q, lr); err != nil {
					continue
				}

				pushCount += len(bulk.Bodies)
			}
		}

		fmt.Printf("Dequeued and pushed %d items\n", pushCount)
	}
}
