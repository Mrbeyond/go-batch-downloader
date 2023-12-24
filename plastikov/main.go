package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	// flag variables
	url := flag.String("url", "", "URL of the file to download")
	output := flag.String("output", "", "Name for the file to download")
	numGoroutines := flag.Int("goroutines", 4, "Number of goroutines to spin when downloading file")
	// retries := flag.Int("retry", 3, "Number of times to retry downloading file")
	timeout := flag.Duration("timeout", 30, "Connection to timeout in 30 seconds")

	flag.Parse()

	// validation for required flags
	if *url == "" || *output == ""{
		fmt.Println("Usage: go-batch-downloader -url <url-to-download-from> -output <output-file>")
	}
	// initialise download instance of the item
	obj := Downloader{URL: *url, outputFile: *output, Goroutines: *numGoroutines, time: *timeout} //Retries: *retries, 

	// download a chunk of the file and check for error
	err := obj.startDownloader()
	if err != nil{
		log.Fatalf("error: download failed %s", err)
	}
	fmt.Println("100% ==> download succesful.")
}

type Downloader struct{
	URL string
	outputFile string
	Goroutines int
	time time.Duration
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

	if headReq.StatusCode != http.StatusOK{
		client := &http.Client{Timeout: d.time}
		resp, err := client.Do(req)
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
	} else {
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
				
				client := &http.Client{Timeout: d.time}
				resp, err := client.Do(req)
				if err != nil{
					fmt.Printf("error: %s", err)
					return
				}
				defer resp.Body.Close()
				
				_, err = io.ReadFull(resp.Body, buffer)
				if err != nil{
					fmt.Printf("error: %s", err)
					return
				}

				fileBuffer.Write(buffer[:])
				out, err := os.Create(d.outputFile)
				if err != nil{
					fmt.Printf("error: %s", err)
					return
				}
				defer out.Close()

				io.Copy(out, &fileBuffer)

				defer wg.Done()
			}()
		}
		wg.Wait()
	}
	return nil
}