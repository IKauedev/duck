package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
)

func ZipCommand() cli.Command {
	return cli.Command{
		Name:        "zip",
		Description: "Compacta arquivos e pastas em ZIP",
		Usage:       "zip <saida.zip> <arquivo|pasta...>",
		Run:         zipCommand,
		Examples: []string{
			"zip backup.zip logs app.env",
			"zip projeto.zip ./src ./README.md",
		},
	}
}

func UnzipCommand() cli.Command {
	return cli.Command{
		Name:        "unzip",
		Description: "Descompacta um arquivo ZIP",
		Usage:       "unzip <arquivo.zip> [destino]",
		Run:         unzipCommand,
		Examples: []string{
			"unzip backup.zip",
			"unzip backup.zip ./restore",
		},
	}
}

func zipCommand(_ cli.Context, args []string) error {
	if len(args) < 2 {
		return cli.UsageError("use: zip <saida.zip> <arquivo|pasta...>")
	}

	output := args[0]
	if !strings.HasSuffix(strings.ToLower(output), ".zip") {
		output += ".zip"
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	for _, source := range args[1:] {
		if err := addZipSource(writer, source); err != nil {
			return err
		}
	}
	fmt.Println("Arquivo criado:", output)
	return nil
}

func addZipSource(writer *zip.Writer, source string) error {
	root, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil {
		return err
	}

	base := filepath.Base(root)
	if !info.IsDir() {
		return addZipFile(writer, root, base, info)
	}

	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(base, rel))
		if rel == "." {
			name = base + "/"
		}
		if entry.IsDir() {
			if !strings.HasSuffix(name, "/") {
				name += "/"
			}
			_, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
			return err
		}
		return addZipFile(writer, path, name, info)
	})
}

func addZipFile(writer *zip.Writer, path string, name string, info os.FileInfo) error {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(name)
	header.Method = zip.Deflate

	dst, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(dst, src)
	return err
}

func unzipCommand(_ cli.Context, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return cli.UsageError("use: unzip <arquivo.zip> [destino]")
	}
	destination := "."
	if len(args) == 2 {
		destination = args[1]
	}
	destination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, 0755); err != nil {
		return err
	}

	reader, err := zip.OpenReader(args[0])
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if err := extractZipFile(file, destination); err != nil {
			return err
		}
	}
	fmt.Println("Arquivo extraido em:", destination)
	return nil
}

func extractZipFile(file *zip.File, destination string) error {
	target := filepath.Join(destination, filepath.Clean(file.Name))
	if !strings.HasPrefix(target, destination+string(os.PathSeparator)) && target != destination {
		return fmt.Errorf("arquivo zip contem caminho inseguro: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(target, file.Mode())
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
