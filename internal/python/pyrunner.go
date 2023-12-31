package python

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/redgoat650/barnacle-net/internal/message"
)

const (
	imagePyFileName    = "image.py"
	identifyPyFileName = "identify.py"
	pythonBin          = "python3"
)

type PyRunner struct {
	scriptsDir string
	mu         *sync.Mutex
}

func NewImagePYRunner(scriptDir string) *PyRunner {
	return &PyRunner{
		scriptsDir: scriptDir,
		mu:         new(sync.Mutex),
	}
}

func (p *PyRunner) RunImagePY(filename string, rotationDeg int, saturation *float64, fitPolicy message.FitPolicy) error {
	if filename == "" {
		return errors.New("invalid file name")
	}

	imagePyPath := p.getImagePyPath()

	cmd := exec.Command(pythonBin, imagePyPath, filename, strconv.Itoa(rotationDeg))

	switch {
	case *saturation < 0:
		cmd.Args = append(cmd.Args, "0")
	case *saturation > 1:
		cmd.Args = append(cmd.Args, "1")
	default:
		cmd.Args = append(cmd.Args, fmt.Sprintf("%.2f", *saturation))
	}

	cmd.Args = append(cmd.Args, string(fitPolicy))

	log.Printf("Executing %q", cmd.String())
	p.mu.Lock()
	defer p.mu.Unlock()

	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("image.py execution ended with error", err)
		log.Println("OUTPUT:", string(b))
		return err
	}

	log.Println("image.py execution complete")
	log.Print(string(b))

	return nil
}

func (p *PyRunner) RunIdentifyPY() ([]byte, error) {
	idPyPath := p.getIdentifyPyPath()

	cmd := exec.Command(pythonBin, idPyPath)

	log.Printf("Executing %q", cmd.String())
	p.mu.Lock()
	defer p.mu.Unlock()

	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("identify.py execution ended with error", err)
		return b, err
	}

	log.Println("identify.py execution complete")
	log.Print(string(b))

	return b, nil
}

func (p *PyRunner) getImagePyPath() string {
	return filepath.Join(p.scriptsDir, imagePyFileName)
}

func (p *PyRunner) getIdentifyPyPath() string {
	return filepath.Join(p.scriptsDir, identifyPyFileName)
}
