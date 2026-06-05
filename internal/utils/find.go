package utils

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
)

type findOptions struct {
	root      string
	term      string
	extension string
	size      sizeFilter
}

type sizeFilter struct {
	enabled bool
	op      byte
	bytes   int64
}

func FindCommand() cli.Command {
	return cli.Command{
		Name:        "find",
		Aliases:     []string{"search"},
		Description: "Busca arquivos por nome, extensao e tamanho",
		Usage:       "find [--path pasta] [--ext extensao] [--size +100MB|-10MB] [termo]",
		Run:         findCommand,
		Examples: []string{
			"find --ext pdf relatorio",
			"find --size +100MB",
			"search --path ./docs --ext md guia",
		},
	}
}

func findCommand(_ cli.Context, args []string) error {
	opts, err := parseFindArgs(args)
	if err != nil {
		return err
	}

	root, err := filepath.Abs(opts.root)
	if err != nil {
		return err
	}
	matches := 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if !matchesFind(opts, entry.Name(), info.Size()) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		fmt.Printf("%s\t%s\n", rel, formatBytes(info.Size()))
		matches++
		return nil
	})
	if err != nil {
		return err
	}
	if matches == 0 {
		fmt.Println("Nenhum arquivo encontrado.")
	}
	return nil
}

func parseFindArgs(args []string) (findOptions, error) {
	opts := findOptions{root: "."}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path", "--dir":
			if i+1 >= len(args) {
				return opts, cli.UsageError(args[i] + " precisa de um valor")
			}
			opts.root = args[i+1]
			i++
		case "--ext":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--ext precisa de um valor")
			}
			opts.extension = normalizeExtension(args[i+1])
			i++
		case "--size":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--size precisa de um valor")
			}
			filter, err := parseSizeFilter(args[i+1])
			if err != nil {
				return opts, err
			}
			opts.size = filter
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return opts, cli.UsageError("opcao invalida para find: " + args[i])
			}
			if opts.term != "" {
				return opts, cli.UsageError("use: find [--path pasta] [--ext extensao] [--size +100MB|-10MB] [termo]")
			}
			opts.term = strings.ToLower(args[i])
		}
	}
	return opts, nil
}

func matchesFind(opts findOptions, name string, size int64) bool {
	if opts.term != "" && !strings.Contains(strings.ToLower(name), opts.term) {
		return false
	}
	if opts.extension != "" && normalizeExtension(filepath.Ext(name)) != opts.extension {
		return false
	}
	if opts.size.enabled {
		switch opts.size.op {
		case '+':
			return size >= opts.size.bytes
		case '-':
			return size <= opts.size.bytes
		default:
			return size == opts.size.bytes
		}
	}
	return true
}

func normalizeExtension(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, ".")
	if value == "" {
		return ""
	}
	return "." + value
}

func parseSizeFilter(value string) (sizeFilter, error) {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return sizeFilter{}, cli.UsageError("--size precisa de um valor")
	}
	filter := sizeFilter{enabled: true}
	if value[0] == '+' || value[0] == '-' {
		filter.op = value[0]
		value = value[1:]
	}

	multiplier := int64(1)
	for _, unit := range []struct {
		suffix string
		value  int64
	}{
		{"GB", 1024 * 1024 * 1024},
		{"G", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"M", 1024 * 1024},
		{"KB", 1024},
		{"K", 1024},
		{"B", 1},
	} {
		if strings.HasSuffix(value, unit.suffix) {
			multiplier = unit.value
			value = strings.TrimSuffix(value, unit.suffix)
			break
		}
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || number < 0 {
		return sizeFilter{}, cli.UsageError("--size invalido, exemplos: +100MB, -10MB, 500KB")
	}
	filter.bytes = int64(number * float64(multiplier))
	return filter, nil
}

func formatBytes(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d%s", size, units[unit])
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}
