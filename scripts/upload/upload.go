/*
 * File: upload.go
 * Project: scripts
 * File Created: Monday, 12th December 2022 8:54:48 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const (
	Filepath  = "/Users/anonymous /Desktop/laprix"
	ProjectID = "63b63a385e259f72da2dd5c0"
	JWT       = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjYzODQzNWI0ZWU5NGZhNGMxNThlNWYwZSIsInVpZCI6IjBjODk5NmY3LTlkOWMtNDViMC04OTNkLTllZmQzY2JiYTRhNCIsImV4cCI6MTY3MjkwMTI1OH0.TDiWLO0wdTL4T4ktCTzSi2JDBU-Cm6_T-C-DUfm1cgA"

	BaseURL = "localhost"
	// BaseURL = "dev.emeraldai-dev.com"
)

var (
	UploadURL = "https://" + BaseURL + "/v1/projects/%s/upload"
)

func upload(filepaths []string, projectId, authJwt string) (map[string]interface{}, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	for _, path := range filepaths {
		err := func(writer *multipart.Writer) error {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			part1, err := writer.CreateFormFile("files", filepath.Base(path))
			if err != nil {
				return err
			}

			if _, err = io.Copy(part1, file); err != nil {
				return err
			}
			return nil
		}(writer)

		if err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	config := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: config}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("POST", fmt.Sprintf(UploadURL, projectId), payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authJwt))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		if res.StatusCode == 401 {
			return nil, fmt.Errorf("unauthorized")
		}
		return nil, fmt.Errorf("non-successful status code=%d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var report map[string]interface{}
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}

	return report, nil
}

func visit(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fileInfo, err := os.Stat(path)
		if err != nil {
			return err
		}

		if !fileInfo.IsDir() {
			*files = append(*files, path)
		}
		return nil
	}
}

func main() {
	var files []string
	if err := filepath.Walk(Filepath, visit(&files)); err != nil {
		panic(err)
	}

	report, err := upload(files, ProjectID, JWT)
	if err != nil {
		panic(err)
	}
	reportBytes, _ := json.Marshal(report)
	fmt.Printf("%s", string(reportBytes))
}
