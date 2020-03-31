package main

import (
	"fmt"
	"os"

	"github.com/akamensky/argparse"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func main() {
	parser := argparse.NewParser("swing", "TODO")

	var files *[]os.File = parser.FileList(
		"f",
		"files",
		os.O_RDONLY,
		os.ModePerm,
		&argparse.Options{Required: false, Help: "List of files to upload"},
	)

	var region *string = parser.String(
		"r",
		"region",
		&argparse.Options{Required: false, Help: "AWS region"},
	)

	var bucket *string = parser.String(
		"b",
		"bucket",
		&argparse.Options{Required: true, Help: "S3 bucket"},
	)

	var swingFile *os.File = parser.File(
		"s",
		"swing-file",
		os.O_RDWR|os.O_CREATE,
		os.ModePerm,
		&argparse.Options{Required: false, Default: "swing.csv", Help: "TODO"},
	)

	var download *bool = parser.Flag(
		"d",
		"download",
		&argparse.Options{Required: false, Help: "TODO"},
	)

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	if files != nil && *download {
		fmt.Println("You can't specify both files and download")
		os.Exit(1)
	}

	// Region set here is overwritten if found in AWS shared credential file
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(*region),
	}))

	manager := Manager{
		SwingFile: swingFile,
		Bucket:    *bucket,
		Session:   sess,
	}

	if *download {
		manager.Download()
	} else {
		manager.Upload(files)
	}
}
