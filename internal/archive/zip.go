package archive

import (
	"archive/zip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CreateProjectZip(projectDir, outputPath string) error {
	return CreateProjectZipWithProgress(projectDir, outputPath, nil)
}

func CreateProjectZipWithProgress(projectDir, outputPath string, progress ProgressFunc) error {
	gitTracked, err := getGitTrackedFiles(projectDir)
	if err != nil {
		return createZipAllFiles(projectDir, outputPath, progress)
	}

	gitDir := filepath.Join(projectDir, ".git")
	if info, e := os.Stat(gitDir); e == nil && info.IsDir() {
		gitFiles, _ := collectDirFiles(projectDir, gitDir)
		gitTracked = append(gitTracked, gitFiles...)
	}

	return createZipFromFiles(projectDir, outputPath, gitTracked, progress)
}

func collectDirFiles(baseDir, dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, err
}

func getGitTrackedFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

func createZipFromFiles(baseDir, outputPath string, files []string, progress ProgressFunc) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	total := len(files)
	processed := 0

	reportProgress := func() {
		if progress == nil || total == 0 {
			return
		}
		processed++
		progress(processed * 100 / total)
	}

	for _, file := range files {
		fullPath := filepath.Join(baseDir, file)

		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}

		zipEntry, err := w.Create(filepath.ToSlash(file))
		if err != nil {
			return err
		}

		f, err := os.Open(fullPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipEntry, f)
		f.Close()
		if err != nil {
			return err
		}

		reportProgress()
	}

	if progress != nil {
		progress(100)
	}

	return nil
}

func createZipAllFiles(baseDir, outputPath string, progress ProgressFunc) error {
	var files []string
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return err
	}

	return createZipFromFiles(baseDir, outputPath, files, progress)
}
