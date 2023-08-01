package python

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

const (
	pythonBin     = "python3"
	imagePyURL    = `https://github.com/pimoroni/inky/blob/master/examples/7color/image.py`
	identifyPyURL = `https://github.com/pimoroni/inky/blob/master/examples/identify.py`
)

type PyRunner struct {
	scriptsDir string
}

func NewImagePYRunner(scriptDir string) *PyRunner {
	return &PyRunner{
		scriptsDir: scriptDir,
	}
}

func (p *PyRunner) RunImagePY(filename string, saturation *float64) error {
	if filename == "" {
		return errors.New("invalid file name")
	}

	err := p.maybeDownloadScript(imagePyURL)
	if err != nil {
		return err
	}

	imagePyPath := p.scriptPathFromURL(imagePyURL)

	cmd := exec.Command(pythonBin, imagePyPath, filename)

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

func (p *PyRunner) RunIdentifyPY() error {
	url := identifyPyURL
	err := p.maybeDownloadScript(url)
	if err != nil {
		return err
	}

	idPyPath := p.scriptPathFromURL(url)

	cmd := exec.Command(pythonBin, idPyPath)

	log.Printf("Executing %q", cmd.String())
	b, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	log.Println("identify.py execution complete")
	log.Print(string(b))

	return nil
}

func (p *PyRunner) scriptPathFromURL(url string) string {
	_, filename := path.Split(imagePyURL)
	return filepath.Join(p.scriptsDir, filename)
}

func (p *PyRunner) maybeDownloadScript(url string) error {
	scriptPath := p.scriptPathFromURL(url)

	_, err := os.Stat(scriptPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return p.download(url)
		}
	}

	// Exists already.
	log.Printf("Found %s locally\n", scriptPath)
	return nil
}

func (p *PyRunner) download(url string) error {
	scriptPath := p.scriptPathFromURL(url)

	log.Printf("Downloading %s to %s", url, scriptPath)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(scriptPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)

	log.Printf("Download of %s complete\n", scriptPath)

	return err
}
