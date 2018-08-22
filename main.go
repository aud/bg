package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type UnsplashLinks struct {
	Links struct{ Download string }
}

type Headers map[string]string

const (
	accessKey           = ""
	lastUpdatedFileName = "last_updated_time.txt"
	imgFileName         = "main.png"
)

var (
	getEnv  = os.Getenv
	command = exec.Command
	create  = os.Create
	ioCopy  = io.Copy
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func get(uri string, client HttpClient, headers ...Headers) []byte {
	req, err := http.NewRequest("GET", uri, nil)

	if err != nil {
		panic(err)
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	return body
}

func fetchRandomImageUri(client HttpClient) string {
	var uri = "https://api.unsplash.com/photos/random"

	headers := Headers{}
	headers["Content-Type"] = "application/json"
	headers["Authorization"] = fmt.Sprintf("Client-ID %s", accessKey)

	resp := get(uri, client, headers)

	var decoded UnsplashLinks
	json.Unmarshal(resp, &decoded)

	return decoded.Links.Download
}

func fetchImage(uri string, client HttpClient) []byte {
	return get(uri, client)
}

func updateDesktopImage() {
	db, err := sql.Open("sqlite3", desktopDbPath())

	if err != nil {
		panic(err)
	}

	defer db.Close()

	_, err = db.Exec("UPDATE DATA SET VALUE = $1", homePathWithFile(imgFileName))

	if err != nil {
		panic(err)
	}
}

func refreshDock() error {
	_, err := command("sh", "-c", "killall Dock").Output()

	return err
}

func homePath() string {
	return getEnv("HOME")
}

func desktopDbPath() string {
	var dbPath = "Library/Application Support/Dock/desktoppicture.db"

	return fmt.Sprintf("%s/%s", homePath(), dbPath)
}

func homePathWithFile(fileName string) string {
	home := fmt.Sprintf("%s/bg", homePath())

	if _, err := os.Stat(home); os.IsNotExist(err) {
		os.Mkdir(home, 0700)
	}

	return fmt.Sprintf("%s/%s", home, fileName)
}

func writeNewFile(contents io.Reader, path string) {
	file, err := create(path)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	_, err = ioCopy(file, contents)

	if err != nil {
		panic(err)
	}
}

func currentTime() int64 {
	return time.Now().Unix()
}

func hasPastTenMinutes() bool {
	return currentTime()-savedTime() > 30
}

func savedTime() int64 {
	file, err := os.Open(homePathWithFile(lastUpdatedFileName))

	if err != nil {
		panic(err)
	}

	// Bit excessive but its cheap
	buf := make([]byte, 50)

	_, err = file.Read(buf)

	if err != nil {
		panic(err)
	}

	// Strip excess bytes and convert to int64
	trimmed := bytes.Trim(buf, "\x00")
	i, err := strconv.ParseInt(string(trimmed), 10, 64)

	if err != nil {
		panic(err)
	}

	return i
}

func saveCurrentTime() {
	file, err := os.Create(homePathWithFile(lastUpdatedFileName))

	if err != nil {
		panic(err)
	}

	defer file.Close()

	_, err = io.WriteString(file, fmt.Sprintf("%v", currentTime()))

	if err != nil {
		panic(err)
	}
}

func main() {
	var operation = func() {
		uri := fetchRandomImageUri(http.DefaultClient)
		img := fetchImage(uri, http.DefaultClient)

		writeNewFile(
			bytes.NewReader(img),
			homePathWithFile(imgFileName),
		)

		updateDesktopImage()
		saveCurrentTime()

		if err := refreshDock(); err != nil {
			fmt.Errorf("Error when trying to refresh dock: %s", err)
		}
	}

	operation()

	for {
		if hasPastTenMinutes() {
			operation()
		}

		time.Sleep(30 * time.Second)
	}

	panic("process killed")
}
