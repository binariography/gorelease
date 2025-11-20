package main

import (
	"cmp"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"runtime"
	"slices"
	"sync"
	"time"
)

var (
	ERRORS int = 0
)

type Bucket struct {
	Errors    map[string]int
	Durations []time.Duration
}

func main() {
	appStartTime := time.Now()
	start("http://localhost:8082/info")
	fmt.Printf("it took %v seconds to run the app\n", time.Since(appStartTime).Seconds())
	fmt.Printf("The number of times it errored %v \n", ERRORS)
}

func start(address string) []Bucket {
	Buckets := []Bucket{}
	numberOfCPUs := runtime.NumCPU()
	numberOfTries := 1000
	var wg sync.WaitGroup
	for i := 0; i < numberOfCPUs; i++ {
		b := Bucket{}
		b.Errors = make(map[string]int)
		wg.Add(1)
		go func(b *Bucket) {
			defer wg.Done()
			call(numberOfTries, i, address, b)

		}(&b)
		Buckets = append(Buckets, b)
	}
	wg.Wait()
	return Buckets
}

func (b *Bucket) addDurationToBucket(d time.Duration) {
	b.Durations = append(b.Durations, d)
}

func (b *Bucket) addErrorToBucket(e string) {
	_, ok := b.Errors[e]
	if ok {
		b.Errors[e]++
		return
	}
	b.Errors[e] = 1
	return
}

func initHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second, // Connection timeout
		KeepAlive: 5 * time.Second,  // Keep-alive period
	}
	t := &http.Transport{
		DialContext: dialer.DialContext,
	}
	t.DisableCompression = true
	t.MaxConnsPerHost = 100
	//t.DisableKeepAlives = true
	t.MaxIdleConns = 100
	t.IdleConnTimeout = 2 * time.Second
	httpClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: t,
	}

	return httpClient

}

func call(n int, nth int, address string, b *Bucket) {
	status := []int{}

	httpClient := initHTTPClient()
	for i := 0; i < n; i++ {
		start := time.Now()

		resp, err := httpClient.Get(address)
		if err != nil {
			log.Print("Error Connecting to server: ", nth, i, err)
			ERRORS++
			b.addErrorToBucket(fmt.Sprint(err))
			b.addDurationToBucket(time.Since(start))
			continue
		}

		defer resp.Body.Close()
		if !slices.Contains(status, resp.StatusCode) {
			_, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Print("Couldn't read the body:", err)
			}
		}
		b.addDurationToBucket(time.Since(start))
	}

	highest := slices.MaxFunc(b.Durations, func(a, b time.Duration) int {
		return cmp.Compare(a, b)
	})
	in := slices.Index(b.Durations, highest)

	slices.SortFunc(b.Durations, func(a, b time.Duration) int {
		return cmp.Compare(a, b)
	})

	fmt.Printf("Maximum is %s with index %d\n", highest, in)
	fmt.Println(b.Durations[len(b.Durations)-100:])
	fmt.Println("Errors:", b.Errors)

}
