package template

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/prompt"
)

func Command() cli.Command {
	return cli.Command{
		Name:        "template",
		Aliases:     []string{"tpl", "scaffold"},
		Description: "Cria projetos a partir de templates prontos",
		Usage:       "template <list|new> [argumentos]",
		Children: []cli.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "Lista templates disponiveis", Usage: "template list", Run: listTemplates},
			{Name: "new", Aliases: []string{"create", "init"}, Description: "Cria projeto a partir de um template", Usage: "template new <tipo> [nome] [--dir <pasta>] [--force]", Run: createTemplate},
		},
		Examples: []string{
			"template list",
			"template new docker api",
			"template new compose web --dir ./web",
			"template new terraform infra",
			"template new jenkins ci",
		},
	}
}

func listTemplates(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("template list nao recebe argumentos")
	}
	fmt.Println("Templates disponiveis:")
	for _, item := range catalog() {
		fmt.Printf("  %-12s %s\n", item.ID, item.Description)
	}
	fmt.Println()
	fmt.Println("Uso: duck template new <tipo> [nome] [--dir pasta]")
	return nil
}

func createTemplate(_ cli.Context, args []string) error {
	opts, err := parseCreateOptions(args)
	if err != nil {
		return err
	}

	def, ok := findTemplate(opts.kind)
	if !ok {
		return cli.UsageError("template invalido: " + opts.kind + ". Use 'duck template list'.")
	}

	targetDir, err := filepath.Abs(opts.dir)
	if err != nil {
		return err
	}

	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		if !isDirEmpty(targetDir) && !opts.force {
			okConfirm, confirmErr := prompt.Confirm("A pasta nao esta vazia. Continuar e sobrescrever arquivos? [s/N] ")
			if confirmErr != nil {
				return confirmErr
			}
			if !okConfirm {
				fmt.Println("Cancelado.")
				return nil
			}
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	} else if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("nao foi possivel criar %s: %w", targetDir, err)
	}

	projectName := sanitizeName(opts.name)
	if projectName == "" {
		projectName = sanitizeName(filepath.Base(targetDir))
	}
	if projectName == "" {
		projectName = "my-app"
	}

	created := 0
	for relPath, content := range def.Files {
		relPath = strings.ReplaceAll(relPath, "{{ProjectName}}", projectName)
		content = render(content, projectName)
		fullPath := filepath.Join(targetDir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}
		if _, err := os.Stat(fullPath); err == nil && !opts.force {
			continue
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("nao foi possivel criar %s: %w", fullPath, err)
		}
		fmt.Println("  created", relPath)
		created++
	}

	if created == 0 {
		fmt.Println("Nenhum arquivo criado. Use --force para sobrescrever arquivos existentes.")
		return nil
	}

	fmt.Println()
	fmt.Println("Projeto", projectName, "criado em:", targetDir)
	if def.NextStep != "" {
		fmt.Println("Proximo passo:", render(def.NextStep, projectName))
	}
	return nil
}

type createOptions struct {
	kind  string
	name  string
	dir   string
	force bool
}

func parseCreateOptions(args []string) (createOptions, error) {
	var opts createOptions
	if len(args) == 0 {
		return opts, cli.UsageError("use: template new <tipo> [nome] [--dir pasta] [--force]")
	}

	opts.kind = strings.ToLower(args[0])
	positional := make([]string, 0, 1)
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--dir precisa de uma pasta")
			}
			opts.dir = args[i+1]
			i++
		case "--force", "-f":
			opts.force = true
		default:
			positional = append(positional, args[i])
		}
	}

	if len(positional) > 1 {
		return opts, cli.UsageError("informe no maximo um nome de projeto")
	}
	if len(positional) == 1 {
		opts.name = positional[0]
	}

	if opts.dir == "" {
		if opts.name != "" {
			opts.dir = opts.name
		} else {
			opts.dir = "."
		}
	} else if opts.name == "" {
		opts.name = filepath.Base(opts.dir)
	}
	return opts, nil
}

func render(content string, projectName string) string {
	replacer := strings.NewReplacer(
		"{{ProjectName}}", projectName,
		"{{ProjectNameLower}}", strings.ToLower(projectName),
	)
	return replacer.Replace(content)
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash && builder.Len() > 0 {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return ""
	}
	re := regexp.MustCompile(`-+`)
	return re.ReplaceAllString(out, "-")
}

func isDirEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) == 0
}
