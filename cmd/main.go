package main

import (
	"fmt"
	"os"

	"github.com/akamensky/argparse"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func main() {
	parser := argparse.NewParser(
		"swing",
		"Swing is used to upload and download files from AWS S3. "+
			"After each upload a CSV, the Swing file, is updated to keep track of a file version,"+
			"it's suggested to commit the Swing file in VCS to keep track of files history."+
			"The Swing file is read when downloading to know which files' version need to be downloaded.",
	)

	var files *[]os.File = parser.FileList(
		"f",
		"files",
		os.O_RDONLY,
		os.ModePerm,
		&argparse.Options{
			Required: false,
			Help:     "List of files to upload",
		},
	)

	var region *string = parser.String(
		"r",
		"region",
		&argparse.Options{
			Required: false,
			Help:     "AWS region of bucket",
		},
	)

	var bucket *string = parser.String(
		"b",
		"bucket",
		&argparse.Options{
			Required: true,
			Help:     "S3 bucket where to upload or download files",
		},
	)

	var swingFile *os.File = parser.File(
		"s",
		"swing-file",
		os.O_RDWR|os.O_CREATE,
		os.ModePerm,
		&argparse.Options{
			Required: false,
			Default:  "swing.csv",
			Help:     "CSV file read to know which files to download or written to after files upload",
		},
	)

	var download *bool = parser.Flag(
		"d",
		"download",
		&argparse.Options{
			Required: false,
			Help:     "Starts download of files found in specified swing file",
		},
	)

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	if files != nil && *download {
		fmt.Println("You can't specify both -f|--files and -d|--download")
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
