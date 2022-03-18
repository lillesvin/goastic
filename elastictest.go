package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	lorem "github.com/drhodes/golorem"
	"github.com/google/uuid"
)

type ElasticTest struct {
	BaseURL string
	Stats   map[string]*ElasticTestStats
}

type ElasticTestStats struct {
	StartTime    time.Time
	NumRequests  int
	Failed       int
	TotalRequest time.Duration
	MinRequest   time.Duration
	MaxRequest   time.Duration
	AvgRequest   time.Duration
}

type ElasticTestRequest struct {
	Type    string
	Request *http.Request
}

type ElasticTestResponse struct {
	Request   *ElasticTestRequest
	Start     time.Time
	Elapsed   time.Duration
	Response  *http.Response
	Completed bool
	Error     error
}

func (t *ElasticTest) PrintStats() {
	for m, stats := range t.Stats {
		fmt.Printf("Mode: %s\n", m)
		fmt.Printf(" - Requests (total):  %d\n", stats.NumRequests)
		fmt.Printf(" - Requests (failed): %d\n", stats.Failed)
		fmt.Printf(" - Request time (total): %.3f s\n", stats.TotalRequest.Seconds())
		fmt.Printf(" - Request time (max.):  %d ms\n", stats.MaxRequest.Milliseconds())
		fmt.Printf(" - Request time (min.):  %d ms\n", stats.MinRequest.Milliseconds())
		fmt.Printf(" - Request time (avg.):  %d ms\n", stats.AvgRequest.Milliseconds())
		fmt.Printf(" - Wall time:            %.3f s\n", time.Now().Sub(stats.StartTime).Seconds())
	}
}

func (t *ElasticTest) HandleResponse(res *ElasticTestResponse) {
	rt := res.Request.Type

	if t.Stats[rt] == nil {
		t.Stats[rt] = &ElasticTestStats{StartTime: time.Now()}
	}

	// Count all requests
	t.Stats[rt].NumRequests += 1

	// Count failed requests
	if res.Completed == false {
		t.Stats[rt].Failed += 1
		fmt.Printf("* %s\n", res.Error)
		return
	}

	// Track max. elapsed
	if res.Elapsed.Milliseconds() > 0 && res.Elapsed > t.Stats[rt].MaxRequest {
		t.Stats[rt].MaxRequest = res.Elapsed
	}

	// Track min. elapsed
	if (t.Stats[rt].NumRequests-t.Stats[rt].Failed) == 1 || (res.Elapsed.Milliseconds() > 0 && res.Elapsed < t.Stats[rt].MinRequest) {
		t.Stats[rt].MinRequest = res.Elapsed
	}

	// Track total elapsed
	t.Stats[rt].TotalRequest += res.Elapsed

	// Update avg. elapsed
	if t.Stats[rt].NumRequests > t.Stats[rt].Failed {
		t.Stats[rt].AvgRequest = t.Stats[rt].TotalRequest / time.Duration((t.Stats[rt].NumRequests - t.Stats[rt].Failed))
	}
}

func (t *ElasticTest) EnqueueRequests(queue chan *ElasticTestRequest, queueKill chan bool, num int, readonly bool) {
	for i := 0; i < num; i++ {
		select {
		case <-queueKill:
			fmt.Println("Closing request queue...")
			close(queue)
			return
		default:
			etr := &ElasticTestRequest{}
			if readonly {
				etr.Request, _ = t.genReadHTTP()
				etr.Type = "read"
			} else {
				if i%2 == 0 {
					etr.Request, _ = t.genReadHTTP()
					etr.Type = "read"
				} else {
					etr.Request, _ = t.genWriteHTTP()
					etr.Type = "write"
				}
			}
			queue <- etr
		}
	}
	close(queue)

}

func (t *ElasticTest) EnsureIndex() {
	client := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}

	r, err := http.NewRequest("PUT", t.BaseURL, nil)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = client.Do(r)
	if err != nil {
		log.Fatalln(err)
	}
}

func (t *ElasticTest) genHTTPRequest(method string, url string, body string) (*http.Request, error) {
	buffer := bytes.NewBuffer([]byte(body))
	r, err := http.NewRequest(method, url, buffer)

	if err != nil {
		return nil, err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", fmt.Sprintf("%s/%s", AppName, AppVersion))
	return r, nil
}

func (t *ElasticTest) genReadHTTP() (*http.Request, error) {
	r, err := t.genHTTPRequest("POST", fmt.Sprintf("%s/_search", t.BaseURL), `{"query":{"match_all":{}}}`)

	if err != nil {
		return nil, err
	}
	return r, nil
}

func (t *ElasticTest) genWriteHTTP() (*http.Request, error) {
	r, err := t.genHTTPRequest("POST", fmt.Sprintf("%s/test", t.BaseURL), fmt.Sprintf(`{"id": "%s", "message": "%s"}`, uuid.New(), lorem.Paragraph(5, 6)))

	if err != nil {
		return nil, err
	}
	return r, nil
}
