package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	flag.IntVar(&numReq, "requests", 10000, "Number of requests to make, set to 0 for MAXINT")
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

	if numReq == 0 {
		// If no number is set, use MaxInt
		numReq = int(^uint(0) >> 1)
	}

	fmt.Printf("Elasticsearch: %s\n", testCase.BaseURL)
	fmt.Printf("Requests:      %d\n", numReq)
	fmt.Printf("Interval:      %d ms\n", interval)
	fmt.Printf("Workers:       %d\n", workers)
	fmt.Printf("ReadOnly:      %t\n\n", readOnly)

	var waitGroup sync.WaitGroup

	workerKill := make(chan bool)
	queueKill := make(chan bool)
	requestChan := make(chan *ElasticTestRequest, 16)
	responseChan := make(chan *ElasticTestResponse)

	// Setup interrupt handler
	interruptHandler(workerKill, queueKill, workers)

	testCase.EnsureIndex()

	go func(numReq int) {
		lastThousand := 0
		counter := 0
		fmt.Println("")
		for res := range responseChan {
			testCase.HandleResponse(res)
			if lastThousand < int(counter/1000) {
				lastThousand++
				fmt.Printf(".")
			}
			counter++
		}
		fmt.Println("")
	}(numReq)

	for i := 0; i < workers; i++ {
		go RequestWorker(&waitGroup, requestChan, responseChan, workerKill, interval)
		waitGroup.Add(1)
	}

	testCase.EnqueueRequests(requestChan, queueKill, numReq, readOnly)

	waitGroup.Wait()

	fmt.Println("\n")

	testCase.PrintStats()
}

func interruptHandler(workerKill, queueKill chan bool, workers int) {
	interruptChan := make(chan os.Signal)

	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-interruptChan

		fmt.Println("\nInterrupt caught. Cleaning up...")
		queueKill <- true
		for i := 0; i < workers; i++ {
			workerKill <- true
		}
	}()
}

func RequestWorker(wg *sync.WaitGroup, in chan *ElasticTestRequest, out chan *ElasticTestResponse, workerKill chan bool, interval int) {
	defer wg.Done()

	client := &http.Client{}

	for etreq := range in {
		select {
		case <-workerKill:
			fmt.Println("Stopping worker.")
			return
		default:
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

			if res.Body != nil {
				res.Body.Close()
			}

			etres.Elapsed = time.Since(etres.Start)

			out <- etres
			time.Sleep(time.Duration(interval) * time.Millisecond)
		}
	}
}
