package cmd

import (
	"bytes"
	"errors"
	"load-tests-cli/internal/entity"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRunLoadAggregatesRequestsAndLimitsConcurrency(t *testing.T) {
	const (
		total       = 6
		concurrency = 2
	)

	var requests atomic.Int64
	var active atomic.Int64
	var maxActive atomic.Int64
	var reached atomic.Int64
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			max := maxActive.Load()
			if current <= max || maxActive.CompareAndSwap(max, current) {
				break
			}
		}

		if reached.Add(1) == concurrency {
			close(release)
		}
		<-release

		n := requests.Add(1)
		switch n % 3 {
		case 1:
			w.WriteHeader(http.StatusOK)
		case 2:
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	report, err := runLoad(server.Client(), server.URL, total, concurrency)
	if err != nil {
		t.Fatalf("runLoad returned an error: %v", err)
	}

	if got := requests.Load(); got != total {
		t.Errorf("request count = %d; want %d", got, total)
	}
	if got := maxActive.Load(); got != concurrency {
		t.Errorf("max concurrency = %d; want %d", got, concurrency)
	}
	if report.TotalRequests != total {
		t.Errorf("total requests = %d; want %d", report.TotalRequests, total)
	}
	if report.SuccessRequests != 2 {
		t.Errorf("successful requests = %d; want 2", report.SuccessRequests)
	}
	for _, status := range []string{"404", "500"} {
		if got := report.OtherStatus[status]; got != total/3 {
			t.Errorf("status %s count = %d; want %d", status, got, total/3)
		}
	}
	if _, exists := report.OtherStatus["200"]; exists {
		t.Error("success status 200 should not be included in other_status")
	}
}

type failingClient struct{}

func (failingClient) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("request failed")
}

func TestRunLoadRecordsRequestErrors(t *testing.T) {
	report, err := runLoad(failingClient{}, "http://example.com", 3, 2)
	if err != nil {
		t.Fatalf("runLoad returned an error: %v", err)
	}

	if got := report.OtherStatus["error"]; got != 3 {
		t.Errorf("error count = %d; want 3", got)
	}
	if report.SuccessRequests != 0 {
		t.Errorf("successful requests = %d; want 0", report.SuccessRequests)
	}
}

func TestDoRequestReturnsHTTPStatusNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	result := doRequest(server.Client(), server.URL)
	if result.status != http.StatusServiceUnavailable {
		t.Errorf("status = %d; want %d", result.status, http.StatusServiceUnavailable)
	}
}

func TestPrintReportUsesTextOutput(t *testing.T) {
	var output bytes.Buffer
	printReport(&output, &entity.Report{
		TotalRequests:   3,
		SuccessRequests: 1,
		FinalTime:       0.25,
		OtherStatus:     map[string]int{"503": 1, "404": 1},
	})

	got := output.String()
	if strings.Contains(got, "{") || strings.Contains(got, "\"total_requests\"") {
		t.Fatalf("report should not be JSON: %q", got)
	}
	if !strings.Contains(got, "Successful requests (HTTP 200): 1") {
		t.Errorf("report does not contain success count: %q", got)
	}
	if !strings.Contains(got, "503: 1") {
		t.Errorf("report does not contain status count: %q", got)
	}
}

func TestPrintReportShowsEmptyStatusSection(t *testing.T) {
	var output bytes.Buffer
	printReport(&output, &entity.Report{})

	if !strings.Contains(output.String(), "Other status codes:") {
		t.Error("report should show the status section when it is empty")
	}
}
