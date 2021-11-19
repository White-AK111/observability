package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// service2 struct for service 2
type service2 struct {
	Namespace        string
	LabelMethod      string
	idCounter        prometheus.Counter
	latencyHistogram *prometheus.HistogramVec
}

func main() {
	wg := sync.WaitGroup{}
	defer wg.Wait()

	// service 1 only metrics
	wg.Add(1)
	go startService1("172.17.0.1:9000", &wg)

	// service 2 metrics + id counter
	wg.Add(1)
	go startService2("172.17.0.1:9001", &wg)
}

// startService1 start service 1
func startService1(addr string, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Printf("Service %d start at %s\n", 1, addr)
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(addr, nil)
}

// startService2 start service 2
func startService2(addr string, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("Service %d start at %s\n", 2, addr)

	srv := service2{
		Namespace:   "service2",
		LabelMethod: "method",
	}

	if err := srv.Init(); err != nil {
		log.Fatal(err)
	}

	if err := srv.Serve(addr); err != nil {
		log.Fatal(err)
	}
}

// processHandler handler for service 2
func (a *service2) processHandler(w http.ResponseWriter, r *http.Request) {
	idValue := r.URL.Query().Get("id")
	startTime := time.Now()

	defer func() {
		a.idCounter.Inc()

		a.latencyHistogram.With(prometheus.Labels{a.LabelMethod: r.Method}).Observe(sinceInMilliseconds(startTime))
	}()

	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond) // имитация работы

	writeResponse(w, http.StatusOK, idValue)
}

// Serve func start http server fo service 2
func (a *service2) Serve(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/process", http.HandlerFunc(a.processHandler)) // /process?id=value
	mux.Handle("/metrics", promhttp.Handler())

	return http.ListenAndServe(addr, mux)
}

// Init prometheus metrics for service 2
func (a *service2) Init() error {
	// prometheus type: counter
	a.idCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: a.Namespace,
		Name:      "id_count",
		Help:      "Count id from query's",
	})

	// prometheus type: histogram
	a.latencyHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: a.Namespace,
		Name:      "latency",
		Help:      "The distribution of the latencies",
		Buckets:   []float64{0, 25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000},
	}, []string{a.LabelMethod})

	prometheus.MustRegister(a.idCounter)
	prometheus.MustRegister(a.latencyHistogram)

	return nil
}

// writeResponse func for write response
func writeResponse(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
	_, _ = w.Write([]byte("\n"))
}

// sinceInMilliseconds func for convert time to milliseconds
func sinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}
