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

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

const (
	accessKey           = ""
	lastUpdatedFileName = "last_updated_time.txt"
	imgFileName         = "main.png"
)

var (
	command     = exec.Command
	create      = os.Create
	getEnv      = os.Getenv
	ioCopy      = io.Copy
	isNotExist  = os.IsNotExist
	mkdir       = os.Mkdir
	stat        = os.Stat
	writeString = io.WriteString
	readFile    = ioutil.ReadFile
)

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

func updateDesktopImage(db *sql.DB) {
	_, err := db.Exec("UPDATE DATA SET VALUE = $1", homePathWithFile(imgFileName))

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

	if _, err := stat(home); isNotExist(err) {
		mkdir(home, 0700)
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

func minutesSinceLastUpdate(min int64) bool {
	return currentTime()-savedTime() > min*60
}

func savedTime() int64 {
	file, err := readFile(homePathWithFile(lastUpdatedFileName))

	if err != nil {
		panic(err)
	}

	i, err := strconv.ParseInt(string(file), 10, 64)

	if err != nil {
		panic(err)
	}

	return i
}

func saveCurrentTime() {
	file, err := create(homePathWithFile(lastUpdatedFileName))

	if err != nil {
		panic(err)
	}

	defer file.Close()

	_, err = writeString(file, fmt.Sprintf("%v", currentTime()))

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

		db, err := sql.Open("sqlite3", desktopDbPath())

		if err != nil {
			panic(err)
		}

		defer db.Close()

		updateDesktopImage(db)
		saveCurrentTime()

		if err := refreshDock(); err != nil {
			fmt.Errorf("Error when trying to refresh dock: %s", err)
		}
	}

	operation()

	for {
		if minutesSinceLastUpdate(10) {
			operation()
		}

		time.Sleep(30 * time.Second)
	}

	panic("process killed")
}
