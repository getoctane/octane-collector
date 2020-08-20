package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/getoctane/octane-collector/util"
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

	// We may want to filter some headers, otherwise we could just use a shallow
	// copy proxyReq.Header = req.Header
	headers := make(http.Header)
	for h, val := range lr.Headers {
		headers[h] = val
	}
	headers.Set("Authorization", clusterKey)

	_, err := util.HttpRequest(lr.Method, url, headers, bytes.NewReader(lr.Body))

	return err
}
