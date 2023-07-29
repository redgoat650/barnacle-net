package python

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const (
	pythonBin  = "python3"
	imagePyURL = `https://github.com/pimoroni/inky/blob/master/examples/7color/image.py`
)

type ImagePYRunner struct {
	scriptLocation string
}

func NewImagePYRunner(fileLoc string) *ImagePYRunner {
	return &ImagePYRunner{
		scriptLocation: fileLoc,
	}
}

func (i *ImagePYRunner) RunImagePY(filename string, saturation *float64) error {
	if filename == "" {
		return errors.New("invalid file name")
	}

	err := i.maybeDownload()
	if err != nil {
		return err
	}

	cmd := exec.Command(pythonBin, filename)

	switch {
	case saturation == nil:
	case *saturation < 0:
		cmd.Args = append(cmd.Args, "0")
	case *saturation > 1:
		cmd.Args = append(cmd.Args, "1")
	default:
		cmd.Args = append(cmd.Args, fmt.Sprintf("%.2f", *saturation))
	}

	log.Printf("Executing %q", cmd.String())
	b, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	log.Println("image.py execution complete")
	log.Print(string(b))

	return nil
}

func (i *ImagePYRunner) maybeDownload() error {
	_, err := os.Stat(i.scriptLocation)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return i.download()
		}
	}

	// Exists already.
	log.Println("Found image.py locally")
	return nil
}

func (i *ImagePYRunner) download() error {
	log.Printf("Downloading %s to %s", imagePyURL, i.scriptLocation)
	resp, err := http.Get(imagePyURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(i.scriptLocation)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)

	log.Printf("Download of image.py complete")

	return err
}
