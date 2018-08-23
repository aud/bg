package main

import (
	"bytes"
	"fmt"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"
	"time"
)

var expectedCommand string
var expectedArgs []string
var expectedPath string
var expectedReader io.Reader
var expectedDir string
var expectedPermission string
var expectedWriteString int64

type MockDefaultClient struct {
	resp string
}

type noopIoReaderCloser struct {
	io.Reader
}

func (noopIoReaderCloser) Close() error {
	return nil
}

func (c *MockDefaultClient) Do(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		Body: noopIoReaderCloser{
			bytes.NewBufferString(c.resp),
		},
	}

	return resp, nil
}

func mockExec(command string, args ...string) *exec.Cmd {
	expectedCommand = command
	expectedArgs = args

	return &exec.Cmd{}
}

func mockGetEnv(key string) string {
	return "home_path"
}

func mockCreate(path string) (*os.File, error) {
	expectedPath = path
	return &os.File{}, nil
}

func mockIoCopy(writer io.Writer, reader io.Reader) (int64, error) {
	expectedReader = reader
	return 0, nil
}

func mockWriteString(writer io.Writer, str string) (int, error) {
	expectedWriteString, _ = strconv.ParseInt(str, 10, 64)
	return 1, nil
}

func mockStat(path string) (os.FileInfo, error) {
	return *new(os.FileInfo), nil
}

func mockIsNotExist(err error) bool {
	return true
}

func mockMkdir(dir string, permission os.FileMode) error {
	expectedDir = dir
	expectedPermission = string(permission)

	return nil
}

func TestRefreshDock(t *testing.T) {
	defer func() {
		command = exec.Command
	}()

	command = mockExec

	refreshDock()

	if "sh" != expectedCommand {
		t.Errorf("Expected %s", expectedCommand)
	}

	if reflect.DeepEqual([]string{"-c", "killall Dock"}, expectedArgs) == false {
		t.Errorf("Expected %s", expectedArgs)
	}
}

func TestHomePath(t *testing.T) {
	getEnv = mockGetEnv

	defer func() {
		getEnv = os.Getenv
	}()

	var expectedValue = "home_path"

	actualValue := homePath()

	if expectedValue != actualValue {
		t.Errorf("Expected %s", actualValue)
	}
}

func TestDesktopDbPath(t *testing.T) {
	getEnv = mockGetEnv

	defer func() {
		getEnv = os.Getenv
	}()

	var expectedPath = "home_path/Library/Application Support/Dock/desktoppicture.db"
	actualPath := desktopDbPath()

	if expectedPath != actualPath {
		t.Errorf("Expected %s", actualPath)
	}
}

func mockReadFile(t string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		expectedPath = path

		return []byte(t), nil
	}
}

func TestCurrentTime(t *testing.T) {
	expectedTime := time.Now().Unix()
	actualTime := currentTime()

	// Handle potential race
	if actualTime-expectedTime > 5 {
		t.Errorf("Expected %v", actualTime)
	}
}

func TestImgFileName(t *testing.T) {
	if "main.png" != imgFileName {
		t.Errorf("Expected %v", imgFileName)
	}
}

func TestLastUpdatedFileName(t *testing.T) {
	if "last_updated_time.txt" != lastUpdatedFileName {
		t.Errorf("Expected %v", lastUpdatedFileName)
	}
}

func TestGet(t *testing.T) {
	mock := &MockDefaultClient{"data"}

	resp := get("https://google.com", mock)

	var expectedData = "data"
	actualData := string(resp)

	if expectedData != actualData {
		t.Errorf("Expected %s", actualData)
	}
}

func TestFetchRandomImageUri(t *testing.T) {
	json := `
		{
			"links": {
				"self": "https://api.unsplash.com/photos/9go97SXw30o",
				"html": "https://unsplash.com/photos/9go97SXw30o",
				"download": "https://unsplash.com/photos/9go97SXw30o/download",
				"download_location": "https://api.unsplash.com/photos/9go97SXw30o/download"
			}
		}
	`

	mock := &MockDefaultClient{json}

	resp := fetchRandomImageUri(mock)

	if "https://unsplash.com/photos/9go97SXw30o/download" != resp {
		t.Errorf("Expected %s", resp)
	}
}

func TestFetchImage(t *testing.T) {
	mock := &MockDefaultClient{"test"}

	resp := fetchImage("https://google.com", mock)

	var expectedData = "test"
	actualData := string(resp)

	if expectedData != actualData {
		t.Errorf("Expected %s", actualData)
	}
}

func TestWriteNewFile(t *testing.T) {
	create = mockCreate
	ioCopy = mockIoCopy

	defer func() {
		create = os.Create
		ioCopy = io.Copy
	}()

	writeNewFile(
		noopIoReaderCloser{bytes.NewBufferString("test")},
		"some/path",
	)

	if expectedPath != "some/path" {
		t.Errorf("Expected %s", expectedPath)
	}

	data := make([]byte, 4)
	expectedReader.Read(data)

	if string(data) != "test" {
		t.Errorf("Expected %s", data)
	}
}

func TestHomePathWithFile(t *testing.T) {
	stat = mockStat
	isNotExist = mockIsNotExist
	mkdir = mockMkdir
	getEnv = mockGetEnv

	defer func() {
		stat = os.Stat
		isNotExist = os.IsNotExist
		mkdir = os.Mkdir
		getEnv = os.Getenv
	}()

	actualPath := homePathWithFile("main.png")
	var expectedPath = "home_path/bg/main.png"

	if expectedPath != actualPath {
		t.Errorf("Expected %s", actualPath)
	}

	if expectedDir != "home_path/bg" {
		t.Errorf("Expected %s", expectedDir)
	}
}

func TestSaveCurrentTime(t *testing.T) {
	create = mockCreate
	writeString = mockWriteString

	defer func() {
		create = os.Create
		writeString = io.WriteString
	}()

	saveCurrentTime()

	expectedTime := time.Now().Unix()

	if expectedTime-expectedWriteString > 5 {
		t.Errorf("Expected %v", expectedWriteString)
	}
}

func TestSavedTime(t *testing.T) {
	readFile = mockReadFile("1535006022")

	defer func() {
		readFile = ioutil.ReadFile
	}()

	st := savedTime()

	if st != int64(1535006022) {
		t.Errorf("Expected %v", st)
	}
}

func TestMinutesSinceLastUpdate(t *testing.T) {
	defer func() {
		readFile = ioutil.ReadFile
	}()

	elevenMinsAgo := time.Now().Unix() - 11*60

	readFile = mockReadFile(fmt.Sprintf("%v", elevenMinsAgo))
	hasPast := minutesSinceLastUpdate(10)

	if hasPast != true {
		t.Errorf("Expected %v", hasPast)
	}

	nineMinsAgo := time.Now().Unix() - 9*60

	readFile = mockReadFile(fmt.Sprintf("%v", nineMinsAgo))
	hasPast = minutesSinceLastUpdate(10)

	if hasPast != false {
		t.Errorf("Expected %v", hasPast)
	}
}

func TestUpdateDesktopImage(t *testing.T) {
	getEnv = mockGetEnv

	defer func() {
		getEnv = os.Getenv
	}()

	db, mock, _ := sqlmock.New()

	defer db.Close()

	expect := mock.ExpectExec("UPDATE DATA SET VALUE = \\$1")
	expect.WithArgs("home_path/bg/main.png")
	expect.WillReturnResult(sqlmock.NewResult(1, 1))

	updateDesktopImage(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unsatisfied expectations %s", err)
	}
}
