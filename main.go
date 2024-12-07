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
	go func() {
		fmt.Println("recordMetrics goroutine started.")
		for {
			// Fixed to 5 minutes
			time.Sleep(300 * time.Second)

			resp, err := http.Get(fetchUrl)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err.Error())
				resp.Body.Close()
				continue
			}
			resp.Body.Close()
			i, err := strconv.Atoi(strings.ReplaceAll(string(body), "\"", ""))
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Println(i)
			oblocUtilization.Set(float64(i))
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
