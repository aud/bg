package main

import (
	"bytes"
	// "fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

var expectedCommand string
var expectedArgs []string
var expectedPath string
var expectedReader io.Reader

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

func fakeExec(command string, args ...string) *exec.Cmd {
	expectedCommand = command
	expectedArgs = args

	return &exec.Cmd{}
}

func SetGetenv(f func(key string) string) {
	getEnv = f
}

func TestRefreshDock(t *testing.T) {
	defer func() {
		command = exec.Command
	}()

	command = fakeExec

	refreshDock()

	if "sh" != expectedCommand {
		t.Errorf("Expected %s", expectedCommand)
	}

	if reflect.DeepEqual([]string{"-c", "killall Dock"}, expectedArgs) == false {
		t.Errorf("Expected %s", expectedArgs)
	}
}

func TestHomePath(t *testing.T) {
	defer func() {
		getEnv = os.Getenv
	}()

	var expectedValue = "value"

	SetGetenv(func(key string) string {
		switch key {
		case "HOME":
			return expectedValue
		default:
			return ""
		}
	})

	actualValue := homePath()

	if expectedValue != actualValue {
		t.Errorf("Expected %s", actualValue)
	}
}

func TestDesktopDbPath(t *testing.T) {
	var expectedPath = "/Users/elliotdohm/Library/Application Support/Dock/desktoppicture.db"
	actualPath := desktopDbPath()

	if expectedPath != actualPath {
		t.Errorf("Expected %s", actualPath)
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

func mockCreate(path string) (*os.File, error) {
	expectedPath = path
	return &os.File{}, nil
}

func mockIoCopy(writer io.Writer, reader io.Reader) (int64, error) {
	expectedReader = reader
	return 0, nil
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
