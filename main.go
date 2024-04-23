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
	// カレントディレクトリを取得
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// マークダウンファイルを走査
	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// ファイルがディレクトリではないかつ、.mdで終わる場合
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
	// ファイルを読み込む
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// フロントマターが存在しない場合
	if !hasFrontMatter(string(content)) {
		// フロントマターを作成
		frontMatter := FrontMatter{
			Tags:  []string{"journal", "driving-school"},
			Date:  creationTime.Format("2006-01-02 15:04:05"),
			Draft: true,
		}

		// YAML形式に変換
		frontMatterBytes, err := yaml.Marshal(frontMatter)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		buf.Write([]byte("---\n"))
		buf.Write([]byte(frontMatterBytes))
		buf.Write([]byte("---\n"))
		buf.Write(content)

		// ファイルを書き込む
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
	// 正規表現でフロントマターが存在するか確認
	re := regexp.MustCompile(`(?s)^---\s*\n(.+\n)*---\s*\n`)
	return re.MatchString(content)
}
