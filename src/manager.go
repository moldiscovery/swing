package swing

import (
	"bufio"
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
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Manager handles upload and download of files from bucket.
// When necessary updates the swing file.
type Manager struct {
	SwingFile *os.File
	SwingDir  string
	Bucket    string
	BatchMode bool
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
func (m *Manager) Upload(files []*os.File) {
	// Filters files not on the same or below level of Swing file.
	// This is done because Swing has been thought to manage files
	// inside a Git directory, the Swing file is meant to live in
	// the root of a repository
	validFiles := make([]*os.File, 0)
	invalidFiles := make([]string, 0)
	for _, f := range files {
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

	fmt.Println("Starting upload")
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
			relFilePath = filepath.ToSlash(relFilePath)

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
		}(*file)
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

	fmt.Println("Upload completed")

	m.updateSwingFile(uploadedFiles)
}

// Verifies the specified file is a subfolder of SwingDir
func (m *Manager) isValidPath(file *os.File) (bool, error) {
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
	files, err := m.readSwingFile()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	toDownload := make([]uploadedFile, 0)
	for _, file := range files {
		path := filepath.Join(m.SwingDir, filepath.FromSlash(file.Path))

		// Check if file exists first of all
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			toDownload = append(toDownload, file)
			continue
		}

		f, err := os.Open(path)
		if err != nil {
			fmt.Printf("Can't open file: %v\n", err)
			continue
		}
		defer f.Close()

		hash := md5.New()
		if _, err := io.Copy(hash, f); err != nil {
			fmt.Printf("Can't calculate MD5 of file: %v\n", err)
			continue
		}
		md5String := hex.EncodeToString(hash.Sum(nil))

		if file.MD5 != md5String {
			toDownload = append(toDownload, file)
		}
	}

	if len(toDownload) > 0 {
		fmt.Println("The following files will be overwritten:")
		for _, file := range toDownload {
			fmt.Println(file.Path)
		}
		reader := bufio.NewReader(os.Stdin)
		if m.BatchMode {
			fmt.Println("Starting download")
		} else {
			fmt.Printf("Do you want to continue? [Y/N]: ")
			text, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Error reading input: %v\n", err)
				os.Exit(1)
			}

			text = strings.ToLower(strings.TrimSpace(text))
			switch text {
			case "yes":
			case "y":
				fmt.Println("Starting download")
			case "no":
			case "n":
				fmt.Println("Aborting download")
				os.Exit(0)
			}
		}
	} else {
		fmt.Println("Nothing to download, files already updated")
	}

	errc := make(chan error)
	filec := make(chan string)

	for _, file := range toDownload {
		go func(f uploadedFile) {
			downloadPath := filepath.Join(m.SwingDir, filepath.FromSlash(f.Path))
			file, err := os.OpenFile(downloadPath, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				errc <- fmt.Errorf("Can't open file: %v", err)
				return
			}

			downloader := s3manager.NewDownloader(m.Session.Copy(
				&aws.Config{
					Region: aws.String(f.Region),
				},
			))
			_, err = downloader.Download(
				file,
				&s3.GetObjectInput{
					Bucket:    aws.String(f.Bucket),
					Key:       aws.String(f.Path),
					VersionId: aws.String(f.VersionID),
				},
			)
			if err != nil {
				errc <- fmt.Errorf("Can't download file: %v", err)
				return
			}
			filec <- downloadPath
		}(file)
	}

	for i := 0; i < len(toDownload); i++ {
		select {
		case err := <-errc:
			fmt.Println(err)
		case f := <-filec:
			fmt.Printf("Downloaded file: %s\n", f)
		}
	}
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

	m.SwingFile.Truncate(0)
	m.SwingFile.Seek(0, 0)

	writer := csv.NewWriter(m.SwingFile)
	writer.Comma = ';'

	// CSV header
	writer.Write([]string{"file", "region", "bucket", "md5", "version_id"})

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

	fmt.Println("Updated Swing file")
}

func (m *Manager) readSwingFile() ([]uploadedFile, error) {
	reader := csv.NewReader(m.SwingFile)
	reader.TrimLeadingSpace = true
	reader.Comma = ';'

	// Reads header and discards it
	reader.Read()

	// Reads the rest of the file
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
