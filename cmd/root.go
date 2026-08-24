package cmd

import (
	"fmt"
	"io"
	"load-tests-cli/internal/entity"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var finalReport entity.Report

var rootCmd = &cobra.Command{
	Use:   "load-tests-cli",
	Short: "Realizar testes de carga em serviços web",
	Long: `O usuário deverá fornecer a URL do serviço, o número total de requisições e a quantidade de chamadas simultâneas.
	Ao final, o sistema deve gerar um relatório detalhado da execução`,
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := getLoads()
		if err != nil {
			return err
		}

		printReport(cmd.OutOrStdout(), report)
		return nil
	},
}

func printReport(output io.Writer, report *entity.Report) {
	fmt.Fprintf(output, "Total requests: %d\n", report.TotalRequests)
	fmt.Fprintf(output, "Successful requests (HTTP 200): %d\n", report.SuccessRequests)
	fmt.Fprintf(output, "Total execution time: %.6f seconds\n", report.FinalTime)
	fmt.Fprintln(output, "Other status codes:")

	nstatus := make([]string, 0, len(report.OtherStatus))
	for status := range report.OtherStatus {
		nstatus = append(nstatus, status)
	}
	sort.Strings(nstatus)
	for _, status := range nstatus {
		fmt.Fprintf(output, "  %s: %d\n", status, report.OtherStatus[status])
	}
}

func getLoads() (*entity.Report, error) {
	return runLoad(http.DefaultClient, URL, TotalRequests, Concurrency)
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type requestResult struct {
	status  int
	success bool
}

func runLoad(client httpDoer, targetURL string, totalRequests int64, concurrency int32) (*entity.Report, error) {
	started := time.Now()
	workers := int64(concurrency)
	if workers > totalRequests {
		workers = totalRequests
	}

	jobs := make(chan struct{})
	results := make(chan requestResult, workers)
	var workerGroup sync.WaitGroup
	workerGroup.Add(int(workers))
	for i := int64(0); i < workers; i++ {
		go func() {
			defer workerGroup.Done()
			for range jobs {
				results <- doRequest(client, targetURL)
			}
		}()
	}

	go func() {
		for i := int64(0); i < totalRequests; i++ {
			jobs <- struct{}{}
		}
		close(jobs)
	}()
	go func() {
		workerGroup.Wait()
		close(results)
	}()

	report := &entity.Report{
		TotalRequests: totalRequests,
		OtherStatus:   make(map[string]int),
	}
	for result := range results {
		if result.success {
			report.SuccessRequests++
			continue
		}

		status := "error"
		if result.status != 0 {
			status = strconv.Itoa(result.status)
		}
		report.OtherStatus[status]++
	}
	report.FinalTime = time.Since(started).Seconds()
	return report, nil
}

func doRequest(client httpDoer, targetURL string) requestResult {
	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return requestResult{}
	}

	response, err := client.Do(request)
	if err != nil || response == nil {
		return requestResult{}
	}
	if response.Body != nil {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}

	return requestResult{
		status:  response.StatusCode,
		success: response.StatusCode == http.StatusOK,
	}
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var URL string
var TotalRequests int64
var Concurrency int32

func init() {
	rootCmd.Flags().StringVarP(&URL, "url", "u", "", "URL do servidor a ser testado")
	rootCmd.Flags().Int64VarP(&TotalRequests, "requests", "r", 1, "Número total de requisições a serem realizadas")
	rootCmd.Flags().Int32VarP(&Concurrency, "concurrency", "c", 1, "Número de chamadas simultâneas")
}
