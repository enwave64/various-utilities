package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Set the path to the USB stick folder
const usbFolder = "H:\\"
const fileTypeConvertFrom = ".wav"

func main() {

	// Channel to communicate errors
	errCh := make(chan error)

	// WaitGroup to synchronize goroutines
	var wg sync.WaitGroup

	// Semaphore to limit the number of concurrent goroutines
	semaphore := make(chan struct{}, 11)

	// Traverse the USB folder recursively
	err := filepath.Walk(usbFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// if info.IsDir() {
		// 	fmt.Printf("%v info isDir() true for \n", path)
		// 	fmt.Printf("%v contains FLAC?: %v\n", path, containsFLAC(path))
		// }

		// fmt.Printf("%v contains FLAC?: %v", path, containsFLAC(path))

		// Check if the folder contains FLAC files
		if info.IsDir() && containsFLAC(path) {

			// fmt.Printf("%v contains FLAC, apparently", path)

			// Increment WaitGroup counter
			wg.Add(1)

			// Acquire semaphore
			semaphore <- struct{}{}

			// Execute conversion in a goroutine
			go func(path string) {
				defer func() {
					// Release semaphore
					<-semaphore
					wg.Done()
				}()

				fmt.Println("Converting files in folder:", path)

				// Convert FLAC files in the album folder
				if err := convertAlbum(path); err != nil {
					errCh <- err
					return
				}
				fmt.Println("Conversion complete for folder:", path)
			}(path)
		}

		return nil
	})

	if err != nil {
		fmt.Println("Error traversing USB folder:", err)
		return
	}

	// Goroutine to handle errors
	go func() {
		for err := range errCh {
			fmt.Println("Error:", err)
		}
	}()

	// Wait for all conversions to complete
	wg.Wait()

	// Close error channel
	close(errCh)

	fmt.Println("Conversion complete.")
}

// containsFLAC checks if a folder contains FLAC files
func containsFLAC(folderPath string) bool {
	fmt.Printf("Checking folder: %v\n", folderPath)
	isContainingFLAC := false
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// fmt.Printf("Checking file: %v\n", path)

		// Check if the file is a FLAC file
		if !info.IsDir() && strings.EqualFold(filepath.Ext(path), fileTypeConvertFrom) {
			fmt.Printf("***********Found FLAC file: %v\n", path)
			isContainingFLAC = true
			return filepath.SkipDir // Stop traversing this folder
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error during traversal: %v\n", err)
	}
	return isContainingFLAC
}

// convertAlbum converts FLAC files in the specified album folder
func convertAlbum(folderPath string) error {
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file is a FLAC file
		if !info.IsDir() && strings.EqualFold(filepath.Ext(path), fileTypeConvertFrom) {
			// Delete existing MP3 file if it exists
			mp3File := strings.TrimSuffix(path, filepath.Ext(path)) + ".mp3"
			if _, err := os.Stat(mp3File); err == nil {
				if err := os.Remove(mp3File); err != nil {
					return fmt.Errorf("failed to delete existing MP3 file: %w", err)
				}
			}

			// Convert FLAC file to 320 kbps MP3
			fmt.Println("We are going to convert for:", path)
			cmd := exec.Command("ffmpeg", "-i", path, "-b:a", "320k", mp3File)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to convert FLAC to MP3: %w", err)
			}

			// Remove the FLAC file to conserve space
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete FLAC file: %w", err)
			}
		}

		return nil
	})

	return err
}
