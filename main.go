package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

var mu sync.RWMutex

var regions = []string{
	"us-east-2",      // 米国東部 (オハイオ)
	"us-east-1",      // 米国東部（バージニア北部）
	"us-west-1",      // 米国西部 (北カリフォルニア)
	"us-west-2",      // 米国西部 (オレゴン)
	"af-south-1",     // アフリカ (ケープタウン)
	"ap-east-1",      // アジアパシフィック (香港)
	"ap-southeast-3", // アジアパシフィック (ジャカルタ)
	"ap-south-1",     // アジアパシフィック (ムンバイ)
	"ap-northeast-3", // アジアパシフィック (大阪)
	"ap-northeast-2", // アジアパシフィック (ソウル)
	"ap-southeast-1", // アジアパシフィック (シンガポール)
	"ap-southeast-2", // アジアパシフィック (シドニー)
	"ap-northeast-1", // アジアパシフィック (東京)
	"ca-central-1",   // カナダ (中部)
	"eu-central-1",   // 欧州 (フランクフルト)
	"eu-west-1",      // 欧州 (アイルランド)
	"eu-west-2",      // 欧州 (ロンドン)
	"eu-south-1",     // ヨーロッパ (ミラノ)
	"eu-west-3",      // 欧州 (パリ)
	"eu-north-1",     // 欧州 (ストックホルム)
	"me-south-1",     // 中東 (バーレーン)
	"sa-east-1",      // 南米 (サンパウロ)
}

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

var arns [][]string
var errs []error

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

func progress(done <-chan struct{}) {
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
		case <-done:
			clear(len(imports[len(imports)-1]))
			fmt.Printf("done impoting %d resources\n", len(arns))
			return
		}
	}
}

func (a *awsController) getAllResources() ([][]string, error) {
	client := resourcegroupstaggingapi.NewFromConfig(a.config)
	arns := make([][]string, 0, 1000)
	var paginationToken *string
	for {
		out, err := client.GetResources(context.Background(), &resourcegroupstaggingapi.GetResourcesInput{
			ResourcesPerPage: aws.Int32(100),
			PaginationToken:  paginationToken,
		})
		if err != nil {
			return nil, err
		}
		for _, v := range out.ResourceTagMappingList {
			arns = append(arns, []string{*v.ResourceARN})
		}
		paginationToken = out.PaginationToken
		if *paginationToken == "" {
			break
		}
	}
	return arns, nil
}

func main() {
	flag.StringVar(&file, "file", "report.csv", "file name to report")
	flag.Parse()

	var wg sync.WaitGroup
	wg.Add(len(regions))
	for _, v := range regions {
		go func(region string) {
			defer wg.Done()
			c, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			ac := &awsController{
				config: c,
			}
			a, err := ac.getAllResources()
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			mu.Lock()
			arns = append(arns, a...)
			mu.Unlock()
		}(v)
	}

	done := make(chan struct{})
	r := localReport{
		path: file,
	}
	go progress(done)

	wg.Wait()
	close(done)

	if err := r.report(arns); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, v := range errs {
		fmt.Fprintln(os.Stderr, v)
		os.Exit(2)
	}

	fmt.Println("all done!")
}
