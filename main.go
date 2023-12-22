package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	chunkSize int64 = 1 << 50
	wg        sync.WaitGroup
)

/*
chunkResp: chunkResp is the type returned by chunkChm channel
from downloadChunk function
*/
type chunkResp struct {
	file  []byte
	index int64
	err   error
}

/*
getFileZise: getFileZise returns content length from the target endpoint.
This to get the target url's file size
*/
func getFileZise(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	size := resp.ContentLength
	if size < 1 {
		return size, fmt.Errorf("Content length is unknown")
	}

	return size, nil
}

func downloadChunk(url string, start int64, end int64, chunkChn chan<- chunkResp, wg *sync.WaitGroup) {
	defer wg.Done()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		chunkChn <- chunkResp{err: err}
		return
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", start, end))

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		chunkChn <- chunkResp{err: err}
		return
	}

	defer resp.Body.Close()
	chunk, err := io.ReadAll(resp.Body)

	if err != nil {
		chunkChn <- chunkResp{err: err}
		return
	}

	// start / int64(chunkSize) return currrent index
	chunkChn <- chunkResp{index: start / chunkSize, file: chunk}
}

func downloadFile(url, destination string) error {

	fileSize, err := getFileZise(url)
	if err != nil {
		return err
	}

	totalChunks := fileSize / chunkSize
	if fileSize%chunkSize != 0 {
		totalChunks++
	}

	chunkChn := make(chan chunkResp, totalChunks)

	for i := int64(0); i < totalChunks; i++ {
		start := i * totalChunks
		end := (i + 1) * chunkSize

		// Check if the last available chunk  is less than the  chunkSize, see line 86
		if end > fileSize {
			end = fileSize
		}

		wg.Add(1)
		fmt.Println(totalChunks, fileSize, "\n\n\n ")
		go downloadChunk(url, start, end, chunkChn, &wg)
	}

	wg.Wait()
	close(chunkChn)

	sortedChunks := make([][]byte, totalChunks)

	for chunk := range chunkChn {
		sortedChunks[chunk.index] = chunk.file
	}

	err = mergeChunks(destination, sortedChunks)

	if err != nil {
		return err
	}

	fmt.Printf("Downloaded %s to %s\n", url, destination)

	return nil
}

func mergeChunks(destination string, chunks [][]byte) error {
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()

	size := int64(len(chunks))
	for i := int64(0); i < size; i++ {
		_, err = file.Write(chunks[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func resolveDestination(destination, filename, url string) (string, error) {
	if filename == "" {
		paths := strings.Split(url, "/")
		filename = paths[len(paths)-1]
	}
	destination = filepath.Join(destination, filename)

	if !filepath.IsAbs(destination) {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory", err)
			return "", err
		}
		destination = filepath.Join(wd, destination)
	}

	return destination, nil
}

func main() {
	startedAt := time.Now()
	url := flag.String("url", "", "Url of file to dowload")
	destination := flag.String("dest", "", "Destination path (optional)")
	filename := flag.String("name", "", "File name (optional)")

	flag.Parse()
	fmt.Println(*url, "\n\n\n ")
	dest, err := resolveDestination(*destination, *filename, *url)
	if err != nil {
		return
	}

	err = downloadFile(*url, dest)
	if err != nil {
		fmt.Println("Error downloading file", err)
	}

	fmt.Println("\n\n Operation took ", time.Since(startedAt).Seconds(), " seconds \n\n ")
}
