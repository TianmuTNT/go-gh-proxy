package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type Config struct {
	WhiteList []string `json:"white_list"`
	BlackList []string `json:"black_list"`
	SizeLimit int64    `json:"size_limit"`
}

var (
	exp1 = regexp.MustCompile(`^(?:https?://)?github\.com/(?P<author>.+?)/(?P<repo>.+?)/(?:releases|archive)/.*$`)
	exp2 = regexp.MustCompile(`^(?:https?://)?github\.com/(?P<author>.+?)/(?P<repo>.+?)/(?:blob|raw)/.*$`)
	exp3 = regexp.MustCompile(`^(?:https?://)?github\.com/(?P<author>.+?)/(?P<repo>.+?)/(?:info|git-).*$`)
	exp4 = regexp.MustCompile(`^(?:https?://)?raw\.(?:githubusercontent|github)\.com/(?P<author>.+?)/(?P<repo>.+?)/.+?/.+$`)
	exp5 = regexp.MustCompile(`^(?:https?://)?gist\.(?:githubusercontent|github)\.com/(?P<author>.+?)/.+?/.+$`)
)

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/favicon.ico", iconHandler)
	http.ListenAndServe(":3426", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Path[1:]

	u = strings.Replace(u, "https:/", "https://", 1)
	u = strings.Replace(u, "http:/", "http://", 1)

	if u == "" {
		index(w, r)
		return
	}

	fmt.Println("\n[URL]", u)

	u, err := url.PathUnescape(u)
	if err != nil {
		http.Error(w, "Failed to decode URL.", http.StatusInternalServerError)
		return
	}

	if m := checkURL(u); m != nil {
		fmt.Printf("[REPO] %s/%s\n", m["author"], m["repo"])
		if allowDownload(m["author"], m["repo"]) {
			proxy(w, r)
		} else {
			http.Error(w, "Download not allowed.", http.StatusForbidden)
		}
	} else {
		http.Error(w, "Invalid input.", http.StatusForbidden)
	}
}


func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func iconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.ico")
}

func checkURL(u string) map[string]string {
	for _, exp := range []*regexp.Regexp{exp1, exp2, exp3, exp4, exp5} {
		match := exp.FindStringSubmatch(u)
		if match != nil {
			result := make(map[string]string)
			for i, name := range exp.SubexpNames() {
				if i > 0 && i <= len(match) {
					result[name] = match[i]
				}
			}
			return result
		}
	}
	return nil
}

func proxy(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Path[1:]
	u = strings.Replace(u, "https:/", "https://", 1)
	u = strings.Replace(u, "http:/", "http://", 1)

	resp, err := http.Get(u)
	if err != nil {
		http.Error(w, "Failed to fetch resource.", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	filename := "downloaded_file.zip"
	if disposition := resp.Header.Get("Content-Disposition"); disposition != "" {
		if matches := regexp.MustCompile(`filename="?([^"]+)"?`).FindStringSubmatch(disposition); len(matches) > 1 {
			filename = matches[1]
		}
	} else {
		parts := strings.Split(u, "/")
		filename = parts[len(parts)-1]
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))

	if _, err := io.Copy(w, resp.Body); err != nil {
		fmt.Println("Error writing response:", err)
		return
	}
}

func iterContent(r *http.Response, chunkSize int) <-chan []byte {
	ch := make(chan []byte)

	go func() {
		defer close(ch)

		for {
			chunk := make([]byte, chunkSize)
			n, err := r.Body.Read(chunk)
			if err != nil {
				if err == io.EOF {
					return
				}
				fmt.Println("Error reading response body:", err)
				return
			}
			ch <- chunk[:n]
		}
	}()

	return ch
}

func allowDownload(author, repo string) bool {
	config := readConfig("config.json")
	if config == nil {
		fmt.Println("Failed to read config.")
		return false
	}

	for _, entry := range config.BlackList {
		if entry == author || entry == author+"/"+repo {
			return false
		}
	}

	if len(config.WhiteList) > 0 {
		for _, entry := range config.WhiteList {
			if entry == author || entry == author+"/"+repo {
				return true
			}
		}
		return false
	}

	return true
}

func readConfig(filename string) *Config {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening config file:", err)
		return nil
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := Config{}
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("Error decoding config file:", err)
		return nil
	}

	return &config
}
