package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/joncrlsn/dque"
)

type proxy struct {
	q *dque.DQue
}

func startProxy(q *dque.DQue) {

	go startQueue(q)

	http.HandleFunc("/", (&proxy{q}).proxyHandler)

	fmt.Println("Starting HTTP server on 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func (p *proxy) proxyHandler(w http.ResponseWriter, req *http.Request) {
	// we need to buffer the body if we want to read it here and send it
	// in the request.
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lr := &LedgerRequest{
		Method:  req.Method,
		Path:    req.RequestURI,
		Body:    body,
		Headers: req.Header,
	}

	// fmt.Printf("Enqueueing %d-byte LedgerRequest %s %s\n", len(body), lr.Method, lr.Path)

	if err := p.q.Enqueue(lr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
	fmt.Fprint(w, `{"acknowledged": true}`)
}

func pushLedgerRequest(lr *LedgerRequest) error {
	// create a new url from the raw RequestURI sent by the client
	url := fmt.Sprintf("%s://%s%s", ledgerScheme, ledgerHost, lr.Path)

	proxyReq, err := http.NewRequest(lr.Method, url, bytes.NewReader(lr.Body))

	// We may want to filter some headers, otherwise we could just use a shallow
	// copy proxyReq.Header = req.Header
	proxyReq.Header = make(http.Header)
	for h, val := range lr.Headers {
		proxyReq.Header[h] = val
	}

	proxyReq.Header.Set("Authorization", clusterKey)

	// fmt.Printf("Pushing %d-byte LedgerRequest %s %s\n", len(lr.Body), lr.Method, lr.Path)

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Status %d from %s: %s", resp.StatusCode, url, string(respBody))
	}

	return nil
}
