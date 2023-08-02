package python

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

const (
	imagePyFileName    = "image.py"
	identifyPyFileName = "identify.py"
	pythonBin          = "python3"
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

	imagePyPath := p.getImagePyPath()

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
		log.Println("image.py execution ended with error", err)
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
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("identify.py execution ended with error", err)
		return nil, err
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
