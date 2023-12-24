package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

func main() {
	file := Downloader{
		URL: "https://z-library.se/dl/1246152/ea6aa2",
		outputFile: "History of Africa.pdf",
		Goroutines: 5,
	}

	err := file.startDownloader(); if err != nil{
		log.Fatalf("%s", err)
	}

	fmt.Println("your file has been downloaded")
}

type Downloader struct{
	URL string
	outputFile string
	Goroutines int
}


func(d *Downloader) startDownloader() error{
	req, err := http.NewRequest("GET", d.URL, nil)
	if err != nil{
		return fmt.Errorf("error: %s", err)
	}

	headReq, err := http.Head(d.URL)
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}

	if headReq.StatusCode == http.StatusOK{
		size := req.Header.Get("Content-length")
		if size == "" {
			return fmt.Errorf("error: Content-Length header is empty")
		}

		fmt.Println("your download has started")
		
		szeInt, err := strconv.Atoi(size)
		if  err != nil{
			return fmt.Errorf("error: %s", err)
		}
		chunk := int64(szeInt/d.Goroutines)
		var (
			startBytes int64
			nextBytes int64
			rangeBytes string
			fileBuffer bytes.Buffer
			wg sync.WaitGroup
		)

		wg.Add(1)
		for i := 0; i < d.Goroutines; i++{
			startBytes = chunk * int64(i)
			nextBytes = startBytes + (chunk - 1)

			rangeBytes = fmt.Sprintf("bytes=%v-%v", startBytes, nextBytes)
			req.Header.Add("Range", rangeBytes)

			go func() {
				
				buffer := make([]byte, chunk)
				
				resp, err := http.DefaultClient.Do(req)
				if err != nil{
					fmt.Errorf("error: %s", err)
				}
				defer resp.Body.Close()
				
				buffer, err = io.ReadAll(resp.Body)
				if err != nil{
					fmt.Errorf("error: %s", err)
				}

				fileBuffer.Write(buffer[:])
				out, err := os.Create(d.outputFile)
				if err != nil{
					fmt.Errorf("error: %s", err)
				}

				io.Copy(out, &fileBuffer)

				defer wg.Done()
			}()
		}
		wg.Wait()
	} else {
		// Fallback to downloading the entire file
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("error: %s", err)
		}
		defer resp.Body.Close()

		out, err := os.Create(d.outputFile)
		if err != nil {
			return fmt.Errorf("error: %s", err)
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Errorf("error: %s", err)
		}
	}
	return nil
}