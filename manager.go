package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Manager handles upload and download of files from bucket.
// When necessary updates the swing file.
type Manager struct {
	SwingFile *os.File
	SwingDir  string
	Bucket    string
	Session   *session.Session
}

type uploadResponse struct {
	File   string
	Output s3manager.UploadOutput
}

// Upload files to Manager Bucket
func (m *Manager) Upload(files *[]os.File) {
	// Filters files not on the same or below level of Swing file.
	// This is done because Swing has been thought to manage files
	// inside a Git directory, the Swing file is meant to live in
	// the root of a repository
	validFiles := make([]os.File, 0)
	invalidFiles := make([]string, 0)
	for _, f := range *files {
		valid, err := m.isValidPath(f)
		if err != nil {
			fmt.Println(err)
		}
		if valid {
			validFiles = append(validFiles, f)
		} else {
			invalidFiles = append(invalidFiles, f.Name())
		}
	}

	// Shows which files are being skipped if any
	if len(invalidFiles) > 0 {
		fmt.Println("Skipping files not in same or below level of Swing file:")
		for _, f := range invalidFiles {
			fmt.Println(f)
		}
	}

	uploader := s3manager.NewUploader(m.Session)
	resc := make(chan uploadResponse)
	errc := make(chan error)

	for _, file := range validFiles {
		go func(f os.File) {
			defer f.Close()
			// TODO: Key should be file's relative path from Swing file
			res, err := uploader.Upload(
				&s3manager.UploadInput{
					Bucket: aws.String(m.Bucket),
					Key:    aws.String(f.Name()),
					Body:   &f,
				},
			)

			if err != nil {
				errc <- err
			} else {
				resc <- uploadResponse{
					File:   f.Name(),
					Output: *res,
				}
			}
		}(file)
	}

	responses := make([]uploadResponse, 0)
	for i := 0; i < len(validFiles); i++ {
		select {
		case err := <-errc:
			fmt.Printf("Upload error: %v\n", err)
		case res := <-resc:
			responses = append(responses, res)
		}
	}
	close(errc)
	close(resc)
}

// Verifies the specified file is a subfolder of SwingDir
func (m *Manager) isValidPath(file os.File) (bool, error) {
	fileAbsPath, err := filepath.Abs(file.Name())
	if err != nil {
		return false, fmt.Errorf("Can't get absolute path to file: %v", err)
	}
	fileDir, _ := filepath.Split(fileAbsPath)
	return strings.Contains(fileDir, m.SwingDir), nil
}

// Download files found in Manager SwingFile
func (m *Manager) Download() {

}

func (m *Manager) updateSwingFile() {

}

func (m *Manager) readSwingFile() {

}
