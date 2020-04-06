package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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

type uploadedFile struct {
	Path      string
	Region    string
	Bucket    string
	MD5       string
	VersionID string
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
	resc := make(chan uploadedFile)
	errc := make(chan error)

	for _, file := range validFiles {
		go func(f os.File) {
			defer f.Close()

			hash := md5.New()
			if _, err := io.Copy(hash, &f); err != nil {
				errc <- fmt.Errorf("Can't calculate MD5 hash of file: %v", err)
				return
			}

			md5String := hex.EncodeToString(hash.Sum(nil))

			f.Seek(0, 0)

			relFilePath, err := m.relativePathToSwingFile(f.Name())
			if err != nil {
				errc <- err
				return
			}

			res, err := uploader.Upload(
				&s3manager.UploadInput{
					Bucket: aws.String(m.Bucket),
					Key:    aws.String(relFilePath),
					Body:   &f,
				},
			)

			if err != nil {
				errc <- err
				return
			}

			resc <- uploadedFile{
				Path:      relFilePath,
				Region:    *m.Session.Config.Region,
				Bucket:    m.Bucket,
				MD5:       md5String,
				VersionID: *res.VersionID,
			}
		}(file)
	}

	uploadedFiles := make([]uploadedFile, 0)
	for i := 0; i < len(validFiles); i++ {
		select {
		case err := <-errc:
			fmt.Printf("Upload error: %v\n", err)
		case res := <-resc:
			uploadedFiles = append(uploadedFiles, res)
		}
	}
	close(errc)
	close(resc)

	m.updateSwingFile(uploadedFiles)
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

func (m *Manager) relativePathToSwingFile(file string) (string, error) {
	// Assuming these return no errors
	fileAbsPath, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("Can't get absolute path to file: %v", err)
	}

	relPath, err := filepath.Rel(m.SwingDir, fileAbsPath)
	if err != nil {
		return "", fmt.Errorf("Can't get relative path to file: %v", err)
	}

	return relPath, nil
}

// Download files found in Manager SwingFile
func (m *Manager) Download() {

}

func (m *Manager) updateSwingFile(files []uploadedFile) {
	savedFiles, err := m.readSwingFile()
	if err != nil {
		// This would be pretty bad, we're not updating the Swing
		// file but at least we uploaded the files correctly
		fmt.Printf("Files are uploaded but Swing file can't be updated: %v\n", err)
		os.Exit(1)
	}

	uniqueFiles := make(map[string]uploadedFile)
	for _, f := range savedFiles {
		uniqueFiles[f.Path] = f
	}
	for _, f := range files {
		uniqueFiles[f.Path] = f
	}

	// We want the files to be sorted when saved
	keys := make([]string, len(uniqueFiles))
	i := 0
	for k := range uniqueFiles {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	writer := csv.NewWriter(m.SwingFile)
	writer.Comma = ';'
	for _, k := range keys {
		line := make([]string, 5)
		line[0] = uniqueFiles[k].Path
		line[1] = uniqueFiles[k].Region
		line[2] = uniqueFiles[k].Bucket
		line[3] = uniqueFiles[k].MD5
		line[4] = uniqueFiles[k].VersionID
		writer.Write(line)
	}
	writer.Flush()
}

func (m *Manager) readSwingFile() ([]uploadedFile, error) {
	reader := csv.NewReader(m.SwingFile)
	reader.TrimLeadingSpace = true
	reader.Comma = ';'

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("Error reading Swing file: %v", err)
	}

	result := make([]uploadedFile, len(lines))
	for i, l := range lines {
		result[i] = uploadedFile{
			Path:      l[0],
			Region:    l[1],
			Bucket:    l[2],
			MD5:       l[3],
			VersionID: l[4],
		}
	}

	return result, nil
}
