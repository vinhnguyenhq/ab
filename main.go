package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	successfulCode = 200
	timeoutCode    = 408
)

type responseInfo struct {
	status   int
	bytes    int64
	duration time.Duration
}

type SummaryInfo struct {
	Hostname          string        `json:"hostname"`
	Port              string        `json:"port"`
	DocumentPath      string        `json:"documentPath"`
	DocumentLength    int           `json:"documentLength"`
	ConcurrencyLevel  int64         `json:"concurrencyLevel"`
	TimeTaken         time.Duration `json:"timeTaken"`
	CompletedRequests int64         `json:"completedRequests"`
	FailedRequesteds  int64         `json:"failedRequests"`
	TotalTransferred  int64         `json:"totalTransferred"`
	Rps               int64         `json:"rps"`
	TimePerRequest    time.Duration `json:"timePerRequest"`
	TransferRate      int64         `json:"transferRate"`
	Requested         int64         `json:"requested"`
	Responded         int64         `json:"responded"`
}

var (
	h                 = &http.Client{}
	summary           = &SummaryInfo{}
	beginBenchmarking = time.Time{}
	endBenchmarking   = time.Time{}
)

func fetch(link string, c chan responseInfo) {
	start := time.Now()

	res, err := h.Get(link)
	if err != nil {

		c <- responseInfo{
			status:   timeoutCode,
			bytes:    0,
			duration: time.Now().Sub(start),
		}

		return
	}

	read, _ := io.Copy(ioutil.Discard, res.Body)

	c <- responseInfo{
		status:   res.StatusCode,
		bytes:    read,
		duration: time.Now().Sub(start),
	}
}

func showSummary(link string) {
	u, _ := url.Parse(link)

	summary.DocumentPath = link
	summary.DocumentLength = len(link)
	summary.Hostname = u.Hostname()
	summary.Port = u.Port()

	endBenchmarking = time.Now()

	summary.TimeTaken = endBenchmarking.Sub(beginBenchmarking)

	sortedSummary, _ := json.MarshalIndent(summary, "", "\t")
	formattedSummary := string(sortedSummary)

	fmt.Println(formattedSummary)
}

func main() {
	requests := flag.Int64("n", 1, "Number of requests to perform")
	concurrency := flag.Int64("c", 1, "Number of multiple requests to make at a time")
	timeout := flag.Int64("t", 3, "Seconds to max. wait for each response")
	timelimit := flag.Float64("l", 10, "Maximum number of seconds to spend for benchmarking")

	flag.Parse()

	if flag.NArg() == 0 || *requests < 0 || *requests < *concurrency {
		flag.PrintDefaults()
		os.Exit(-1)
	}

	h.Timeout = time.Duration(*timeout * 1e9)

	beginBenchmarking = time.Now()

	link := flag.Arg(0)
	c := make(chan responseInfo)

	for i := int64(0); i < *concurrency; i++ {
		summary.Requested++
		go fetch(link, c)
	}

	for response := range c {
		if summary.Requested < *requests {
			summary.Requested++
			go fetch(link, c)
		}

		summary.Responded++

		if response.status == successfulCode {
			summary.CompletedRequests++
		} else {
			summary.FailedRequesteds++
		}

		if time.Now().Sub(beginBenchmarking).Seconds() > *timelimit {
			fmt.Println("time's up")
			showSummary(link)
			break
		}

		if summary.Requested == summary.Responded {
			showSummary(link)
			break
		}
	}
}
