package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type queryParams map[string]string
type headers map[string]string

type TestResult struct {
	SucceededRequests int     `json:"succeededRequests"`
	FailedRequests    int     `json:"failedRequests"`
	TotalRequests     int     `json:"totalRequests"`
	StartTime         string  `json:"startTime"`
	EndTime           string  `json:"endTime"`
	TotalTime         string  `json:"totalTime"`
	RequestsPerSecond float64 `json:"requestsPerSecond"`
	DataTransferred   float64 `json:"dataTransferedInMB"`
}

type TestSettings struct {
	Amount  int         `json:"amount"`
	Worker  int         `json:"worker"`
	Host    string      `json:"host"`
	Query   queryParams `json:"query"`
	Headers headers     `json:"headers"`
}

type TestRequest struct {
	Status         int     `json:"status"`
	ResponseTime   int64   `json:"responseTime"`
	ResponseBodyMB float64 `json:"responseBodyMB"`
}

type TestOutput struct {
	Result   TestResult    `json:"result"`
	Settings TestSettings  `json:"settings"`
	Requests []TestRequest `json:"request"`
}

func (p *queryParams) String() string {
	var params []string
	for key, value := range *p {
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(params, "&")
}

func (p *queryParams) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format for the query parameter: %s", value)
	}
	(*p)[parts[0]] = parts[1]
	return nil
}

func (h *headers) String() string {
	var headerList []string
	for key, value := range *h {
		headerList = append(headerList, fmt.Sprintf("%s: %s", key, value))
	}
	return strings.Join(headerList, ", ")
}

func (h *headers) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format for the header: %s", value)
	}
	(*h)[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	return nil
}

func sendRequest(client *http.Client, requestMethod, endpoint string, customHeaders headers, params queryParams, resultsChan chan<- TestRequest, progressChan chan<- struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	startTime := time.Now()
	req, err := http.NewRequest(requestMethod, endpoint, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		resultsChan <- TestRequest{Status: 0, ResponseTime: 0, ResponseBodyMB: 0}
		return
	}

	for key, value := range customHeaders {
		req.Header.Add(key, value)
	}

	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		resultsChan <- TestRequest{Status: 0, ResponseTime: 0, ResponseBodyMB: 0}
		return
	}

	responseTime := time.Since(startTime).Milliseconds()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		resp.Body.Close()
		resultsChan <- TestRequest{Status: resp.StatusCode, ResponseTime: responseTime, ResponseBodyMB: 0}
		return
	}

	resp.Body.Close()
	resultsChan <- TestRequest{
		Status:         resp.StatusCode,
		ResponseTime:   responseTime,
		ResponseBodyMB: float64(len(body)) / (1 << 20),
	}
	progressChan <- struct{}{}
}

func main() {
	endpoint := flag.String("endpoint", "", "API endpoint to test")
	requestMethod := flag.String("method", "GET", "HTTP request method")
	amount := flag.Int("amount", 1, "Number of requests to send")
	worker := flag.Int("worker", 10, "Number of concurrent workers")
	outputFile := flag.String("output", "results.json", "Output JSON file")
	var params queryParams = make(map[string]string)
	flag.Var(&params, "query", "Query parameters in the format key=value. Can be used multiple times.")
	var customHeaders headers = make(map[string]string)
	flag.Var(&customHeaders, "header", "Custom headers in the format key:value. Can be used multiple times.")

	flag.Parse()

	if *endpoint == "" {
		fmt.Println("Provide an endpoint to test with the -endpoint flag")
		os.Exit(1)
	}

	resultsChan := make(chan TestRequest, *amount)
	progressChan := make(chan struct{}, *amount)
	var wg sync.WaitGroup
	var succeededRequests, failedRequests int

	startTime := time.Now()

	client := &http.Client{}
	go showProgress(progressChan, *amount)

	for i := 0; i < *amount; i++ {
		wg.Add(1)
		go sendRequest(client, *requestMethod, *endpoint, customHeaders, params, resultsChan, progressChan, &wg)
		if i%*worker == 0 {
			wg.Wait() // Wait for a batch of workers to finish before launching new ones
		}
	}
	wg.Wait() // Ensure all goroutines have finished
	close(resultsChan)
	close(progressChan)

	results := make([]TestRequest, 0, *amount)
	for result := range resultsChan {
		if result.Status > 0 {
			succeededRequests++
		} else {
			failedRequests++
		}
		results = append(results, result)
	}

	endTime := time.Now()
	totalTime := endTime.Sub(startTime).Seconds()

	output := TestOutput{
		Result: TestResult{
			SucceededRequests: succeededRequests,
			FailedRequests:    failedRequests,
			TotalRequests:     *amount,
			StartTime:         startTime.Format(time.RFC3339),
			EndTime:           endTime.Format(time.RFC3339),
			TotalTime:         fmt.Sprintf("%.2f seconds", totalTime),
			RequestsPerSecond: float64(succeededRequests) / totalTime,
			DataTransferred:   calculateDataTransferred(results),
		},
		Settings: TestSettings{
			Amount:  *amount,
			Worker:  *worker,
			Host:    *endpoint,
			Query:   params,
			Headers: customHeaders,
		},
		Requests: results,
	}

	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println("Error while marshaling JSON:", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(*outputFile, outputJSON, 0644)
	if err != nil {
		fmt.Println("Error while writing to JSON file:", err)
		os.Exit(1)
	}

	fmt.Println("Results written to", *outputFile)
}

func calculateDataTransferred(requests []TestRequest) float64 {
	var totalDataTransferred float64
	for _, request := range requests {
		totalDataTransferred += request.ResponseBodyMB
	}
	return totalDataTransferred
}

func showProgress(progressChan <-chan struct{}, total int) {
	var completed int
	for range progressChan {
		completed++
		fmt.Printf("\rProgress: %d/%d", completed, total)
		if completed == total {
			fmt.Println("\nAll requests completed.")
			break
		}
	}
}
