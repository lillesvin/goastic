package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	AppName     string = "Goastic"
	AppVersion  string = "0.0.2"
	numReq      int
	interval    int
	workers     int
	readOnly    bool
	es_basename string
	version     bool
)

func init() {
	flag.StringVar(&es_basename, "baseurl", "http://localhost:9200/test", "Base URL of Elasticsearch")
	flag.IntVar(&numReq, "requests", 10000, "Number of requests to make")
	flag.IntVar(&interval, "interval", 5, "Interval between requests in ms")
	flag.IntVar(&workers, "workers", 2, "Number of parallel workers to run")
	flag.BoolVar(&readOnly, "readonly", false, "Only test reads")
	flag.BoolVar(&version, "version", false, "Show version and exit")
	flag.Parse()
}

func main() {
	if version {
		fmt.Printf("%s %s\n", AppName, AppVersion)
		os.Exit(0)
	}

	testCase := &ElasticTest{
		BaseURL: es_basename,
		Stats:   make(map[string]*ElasticTestStats),
	}

	fmt.Printf("Elasticsearch: %s\n", testCase.BaseURL)
	fmt.Printf("Requests:      %d\n", numReq)
	fmt.Printf("Interval:      %d ms\n", interval)
	fmt.Printf("Workers:       %d\n", workers)
	fmt.Printf("ReadOnly:      %t\n\n", readOnly)

	var waitGroup sync.WaitGroup

	requestChan := make(chan *ElasticTestRequest)
	responseChan := make(chan *ElasticTestResponse)

	testCase.EnsureIndex()

	go func(numReq int) {
		percent := 0
		new_percent := 0
		counter := 1

		fmt.Println("")
		for res := range responseChan {
			new_percent = int(math.Round((float64(counter) / float64(numReq)) * float64(100)))
			if new_percent > percent {
				fmt.Print(".")
				percent = new_percent
			}
			testCase.HandleResponse(res)
			counter += 1
		}
	}(numReq)

	for i := 0; i < workers; i++ {
		go RequestWorker(&waitGroup, requestChan, responseChan, interval)
		waitGroup.Add(1)
	}

	testCase.EnqueueRequests(requestChan, numReq, readOnly)

	waitGroup.Wait()

	fmt.Println("\n")

	testCase.PrintStats()
}

func RequestWorker(wg *sync.WaitGroup, in chan *ElasticTestRequest, out chan *ElasticTestResponse, interval int) {
	defer wg.Done()

	client := &http.Client{}

	for etreq := range in {
		etres := &ElasticTestResponse{
			Start:   time.Now(),
			Request: etreq,
		}

		res, err := client.Do(etreq.Request)

		etres.Response = res
		etres.Completed = true

		if err != nil {
			etres.Completed = false
			etres.Response = nil
			etres.Error = err
		}

		etres.Elapsed = time.Since(etres.Start)

		out <- etres
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}
