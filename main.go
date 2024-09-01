package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	oblocUtilization = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "obloc_utilization_percent",
		Help: "The current O'Bloc utilization",
	})

	fetchUrl = "https://obloc.ch/_cmsbox_backends_/obloc/guestcounter/"
)

func recordMetrics() {
	// Record metrics every 5 minutes
	go func() {
		for {
			resp, err := http.Get(fetchUrl)
			if err != nil {
				fmt.Println(fmt.Errorf(err.Error()))
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(fmt.Errorf(err.Error()))
				resp.Body.Close()
			}
			resp.Body.Close()
			i, err := strconv.Atoi(strings.ReplaceAll(string(body), "\"", ""))
			if err != nil {
				fmt.Println(fmt.Errorf(err.Error()))
			}
			fmt.Println(i)

			oblocUtilization.Set(float64(i))
			// Fixed to 5 minutes
			time.Sleep(300 * time.Second)
		}
	}()
}

func main() {

	val, present := os.LookupEnv("FETCH_URL")
	if present {
		fetchUrl = val
	}

	// Start actual gathering
	recordMetrics()

	// Publish prometheus endpoints
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8081", nil)
}
