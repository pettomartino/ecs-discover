package main

import (
	"ecs-export/awsService"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func parseFlags() (*string, *string, *string, *string, bool) {
	cluster := flag.String("c", "", "AWS Cluster arn")
	region := flag.String("r", "", "AWS Region")
	entryPoints := flag.String("ep", "", "Regex that should match to a container name(separated by comma)")
	templ := flag.String("t", "", "Path to template")
	flag.Parse()

	missing := 0
	if *cluster == "" {
		fmt.Println("cluster required")
		missing++
	}

	if *region == "" {
		fmt.Println("region required")
		missing++
	}

	if *entryPoints == "" {
		fmt.Println("entry point required")
		missing++
	}

	if *templ == "" {
		fmt.Println("template required")
		missing++
	}

	if missing > 0 {
		return nil, nil, nil, nil, false
	}

	return cluster, region, entryPoints, templ, true
}

func main() {
	cluster, region, entryPoints, templ, ok := parseFlags()

	if !ok {
		return
	}

	if _, err := os.Stat(*templ); os.IsNotExist(err) {
		fmt.Println("template file not found")
		return
	}

	awsI := awsService.NewAwsService(*cluster, strings.Split(*entryPoints, ","))

	session := session.New(&aws.Config{Region: aws.String(*region)})
	entries, err := awsI.GetEntries(ecs.New(session), ec2.New(session))

	if err != nil {
		panic(err)
	}

	tpl := template.New("template")
	templContent, err := ioutil.ReadFile(*templ)

	ctx := struct {
		Entries []awsService.Entry
	}{
		entries,
	}
	tpl = template.Must(tpl.Parse(string(templContent)))
	tpl.Execute(os.Stdout, ctx)
}
