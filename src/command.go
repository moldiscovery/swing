package swing

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const usage string = `Usage: swing [-h|--help]  [-y|--yes] [-r|--region "<value>"] -b|--bucket
		"<value>" [-s|--swing-file <file>] 
		[-d|--download] <files...>

	Swing is used to upload and download files from AWS S3.
	After each upload a CSV, the Swing file, is updated to keep track of a file version.
	It's suggested to commit the Swing file in VCS to keep track of files history.
	The Swing file is read when downloading to know which files' version need to be downloaded.
`

func Run() {
	var region, bucket, swingFilePath string
	var download, help, version, confirm bool

	const regionDefault string = ""
	const regionUsage string = "AWS region of bucket"
	flag.StringVar(&region, "region", regionDefault, regionUsage)
	flag.StringVar(&region, "r", regionDefault, regionUsage+" (shorthand)")

	const bucketDefault string = ""
	const bucketUsage string = "S3 bucket where to upload or download files"
	flag.StringVar(&bucket, "bucket", bucketDefault, bucketUsage)
	flag.StringVar(&bucket, "b", bucketDefault, bucketUsage+" (shorthand)")

	const swingFilePathDefault string = "swing.csv"
	const swingFilePathUsage string = "CSV file read to know which files to download or written to after files upload"
	flag.StringVar(&swingFilePath, "swing-file", swingFilePathDefault, swingFilePathUsage)
	flag.StringVar(&swingFilePath, "s", swingFilePathDefault, swingFilePathUsage+" (shorthand)")

	const downloadDefault bool = false
	const downloadUsage string = "Starts download of files found in specified swing file"
	flag.BoolVar(&download, "download", downloadDefault, downloadUsage)
	flag.BoolVar(&download, "d", downloadDefault, downloadUsage+" (shorthand)")

	const helpDefault bool = false
	const helpUsage string = "Prints help text"
	flag.BoolVar(&help, "help", helpDefault, helpUsage)
	flag.BoolVar(&help, "h", helpDefault, helpUsage+" (shorthand)")

	const versionDefault bool = false
	const versionUsage string = "Prints Swing version"
	flag.BoolVar(&version, "version", versionDefault, versionUsage)
	flag.BoolVar(&version, "v", versionDefault, versionUsage+" (shorthand)")

	const confirmDefault bool = false
	const confirmUsage string = "Assume yes to all questions"
	flag.BoolVar(&confirm, "yes", confirmDefault, confirmUsage)
	flag.BoolVar(&confirm, "y", confirmDefault, confirmUsage+" (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Swing version %s\n", CurrentVersion)
		fmt.Fprintln(flag.CommandLine.Output(), usage)
		fmt.Fprintln(flag.CommandLine.Output(), "Arguments:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	flag.Parse()

	if help || flag.NFlag() == 0 {
		flag.Usage()
	}

	if version {
		fmt.Printf("Swing version %s\n", CurrentVersion)
		os.Exit(0)
	}

	if len(bucket) == 0 && !download {
		fmt.Println("[-b|--bucket] is required")
		os.Exit(1)
	}

	filesPaths := flag.Args()
	if len(filesPaths) > 0 && download {
		fmt.Println("You can't specify both files and -d|--download")
		os.Exit(1)
	}

	swingAbsPath, err := filepath.Abs(swingFilePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	swingDir, _ := filepath.Split(swingAbsPath)

	sess, err := Authorize(region)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	swingFile, err := os.OpenFile(swingAbsPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("Can't open Swing file: %v\n", err)
		os.Exit(1)
	}

	manager := Manager{
		SwingFile: swingFile,
		SwingDir:  swingDir,
		Bucket:    bucket,
		BatchMode: confirm,
		Session:   sess,
	}

	files := make([]*os.File, 0)
	for _, filePath := range filesPaths {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Error opening file %s, skipping: %v", filePath, err)
			continue
		}
		files = append(files, file)
	}

	if download {
		manager.Download()
	} else {
		manager.Upload(files)
	}
}
