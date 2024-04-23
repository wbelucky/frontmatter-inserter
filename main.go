package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v2"
)

type FrontMatter struct {
	Tags  []string `yaml:"tag"`
	Date  string   `yaml:"date"`
	Draft bool     `yaml:"draft"`
}

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			var stat unix.Statx_t
			if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &stat); err != nil {
				fmt.Printf("failed to get unix.Statx of %s: %w\n", path, err)
				return nil
			}
			fmt.Printf("%#v\n", stat)
			creationTime := time.Unix(int64(stat.Btime.Sec), int64(stat.Btime.Nsec))

			if err := processMarkdownFile(path, creationTime); err != nil {
				fmt.Printf("Error processing %s: %w\n", path, err)
				return nil
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking directory:", err)
	}
}

func processMarkdownFile(filePath string, creationTime time.Time) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if !hasFrontMatter(string(content)) {
		frontMatter := FrontMatter{
			Tags:  []string{"journal", "driving-school"},
			Date:  creationTime.Format("2006-01-02 15:04:05"),
			Draft: true,
		}

		frontMatterBytes, err := yaml.Marshal(frontMatter)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		buf.Write([]byte("---\n"))
		buf.Write([]byte(frontMatterBytes))
		buf.Write([]byte("---\n"))
		buf.Write(content)

		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to open: %w")
		}
		_, err = buf.WriteTo(f)

		if err1 := f.Close(); err1 != nil && err == nil {
			err := err1
			return fmt.Errorf("failed to close %s: %w\n", filePath, err)
		}
		if err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}

		fmt.Println("Front matter added to", filePath)
	}

	return nil
}

func hasFrontMatter(content string) bool {
	re := regexp.MustCompile(`(?s)^---\s*\n(.+\n)*---\s*\n`)
	return re.MatchString(content)
}
