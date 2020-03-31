package main

import (
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
)

type Manager struct {
	SwingFile *os.File
	Bucket    string
	Session   *session.Session
}

func (m *Manager) Upload(files *[]os.File) {

}

func (m *Manager) Download() {

}

func (m *Manager) updateSwingFile() {

}

func (m *Manager) readSwingFile() {

}
