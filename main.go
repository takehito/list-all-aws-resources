package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

type awsController struct {
	config aws.Config
}

var file string

var imports = []string{
	"importing",
	"importing.",
	"importing..",
	"importing...",
}

// クリアしたい文字数
func clear(num int) {
	fmt.Printf("\r%s\r", strings.Repeat(" ", num))
}

type localReport struct {
	path string
}

func (r *localReport) report(arns [][]string) error {
	buf := bytes.NewBuffer([]byte(""))
	w := csv.NewWriter(buf)
	if err := w.WriteAll(arns); err != nil {
		return err
	}
	return os.WriteFile(r.path, buf.Bytes(), os.FileMode(0666))
}

func printProgress(report localReport, arnsCh <-chan [][]string, errCh <-chan error) error {
	count := 0
	for {
		select {
		case <-time.Tick(time.Second):
			i := count % len(imports)
			if i == 0 {
				clear(len(imports[len(imports)-1]))
			}
			status := imports[i]
			fmt.Printf("%s", status)
			fmt.Print("\r")
			count++
		case arns := <-arnsCh:
			if err := report.report(arns); err != nil {
				return err
			}

			clear(len(imports[len(imports)-1]))
			fmt.Println("done")

			return nil
		case err := <-errCh:
			return err
		}
	}
}

func (a *awsController) getAllResources() ([][]string, error) {
	client := resourcegroupstaggingapi.NewFromConfig(a.config)
	arns := make([][]string, 0, 1000)
	requestCount := 0
	for {
		out, err := client.GetResources(context.Background(), &resourcegroupstaggingapi.GetResourcesInput{
			ResourcesPerPage: aws.Int32(100),
		})
		if err != nil {
			return nil, err
		}
		for _, v := range out.ResourceTagMappingList {
			arns = append(arns, []string{*v.ResourceARN})
		}
		if out.PaginationToken == nil {
			break
		}
		if requestCount <= 0 {
			requestCount = 3
			time.Sleep(3 * time.Second)
		} else {
			requestCount--
		}
	}
	return arns, nil
}

func main() {
	flag.StringVar(&file, "file", "report.csv", "file name to report")
	flag.Parse()

	c, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ac := &awsController{
		config: c,
	}

	arnsCh := make(chan [][]string)
	errorCh := make(chan error)
	go func(arnsCh chan<- [][]string) {
		a, err := ac.getAllResources()
		if err != nil {
			errorCh <- err
			return
		}
		arnsCh <- a
	}(arnsCh)

	r := localReport{
		path: file,
	}
	if err := printProgress(r, arnsCh, errorCh); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
