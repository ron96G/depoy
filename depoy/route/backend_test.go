package route

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var promMetricString = `
# Finally a summary, which has a complex representation, too:
# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 3102
rpc_duration_seconds{quantile="0.05"} 3272
rpc_duration_seconds{quantile="0.5"} 4773
rpc_duration_seconds{quantile="0.9"} 9001
rpc_duration_seconds{quantile="0.99"} 76656
rpc_duration_seconds_sum 1.7560473e+07
rpc_duration_seconds_count 2693
`

var testURL string

var scrapeMetrics = []*ScrapeMetric{
	&ScrapeMetric{
		Name:          "rpc_duration_seconds_sum",
		Threshhold:    2,
		ScrapeCounter: 0,
	},
	&ScrapeMetric{
		Name:          "rpc_duration_seconds_count",
		Threshhold:    3000,
		ScrapeCounter: 0,
	},
}

func init() {
	// start test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "/metrics") {
			w.Write([]byte(promMetricString))

		} else {
			w.WriteHeader(404)
		}
	}))
	testURL = ts.URL + "/metrics"

}

func Test_NewBackend(t *testing.T) {

	backend := NewBackend("test", testURL)
	backend.SetupScrape(testURL, 5*time.Second)

	backend.AddScrapeMetric("rpc_duration_seconds_count", 3000)
	backend.AddScrapeMetric("rpc_duration_seconds_sum", 2)

	// give it some time to run once
	time.Sleep(6 * time.Second)
}

func Test_getRowFromBodyInt(t *testing.T) {
	r := strings.NewReader(promMetricString)
	val, err := getRowFromBody(r, "rpc_duration_seconds_count")
	if err != nil {
		t.Error(err.Error())
	}

	expected := float64(2693)
	if val != expected {
		t.Errorf("Value not equal. Expected %f. Got %f", expected, val)
	}
}

func Test_getRowFromBodyFloat(t *testing.T) {
	r := strings.NewReader(promMetricString)

	val, err := getRowFromBody(r, "rpc_duration_seconds_sum")
	if err != nil {
		t.Error(err.Error())
	}

	expected := 1.7560473e+07
	if val != expected {
		t.Errorf("Value not equal. Expected %f. Got %f", expected, val)
	}
}

func Test_doScrape(t *testing.T) {
	backend := NewBackend("test", testURL)

	backend.SetupScrape(testURL, 5*time.Second)
	backend.AddScrapeMetric("rpc_duration_seconds_count", 3000)
	backend.AddScrapeMetric("rpc_duration_seconds_sum", 2)
	backend.DoScrape()
	backend.DoScrape()

	time.Sleep(6 * time.Second)
	// this must be 4 ... 2 from scrapeJob, 2 from DoScrape
	exptected := 4
	if backend.ScrapeMetrics[0].ScrapeCounter != exptected {
		t.Errorf("ScrapeCounter should be %d and not %d", exptected, backend.ScrapeMetrics[0].ScrapeCounter)
	}
}