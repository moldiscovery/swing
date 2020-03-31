package main

import (
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
)

// Manager handles upload and download of files from bucket.
// When necessary updates the swing file.
type Manager struct {
	SwingFile *os.File
	Bucket    string
	Session   *session.Session
}

// Upload files to Manager Bucket
func (m *Manager) Upload(files *[]os.File) {

}

// Download files found in Manager SwingFile
func (m *Manager) Download() {

}

func (m *Manager) updateSwingFile() {

}

func (m *Manager) readSwingFile() {

}
