package mongodb

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
)

type service struct {
	bin    string // mongosh ou mongo
	runner runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	bin := resolveBin(cfg)
	svc := service{bin: bin, runner: run}
	return cli.Command{
		Name:        "mongo",
		Aliases:     []string{"mongodb", "mg"},
		Description: "Gerencia e interage com MongoDB",
		Usage:       "mongo <comando> [opcoes]",
		Children: []cli.Command{
			{
				Name:        "status",
				Description: "Verifica se o MongoDB está acessível e mostra informações do servidor",
				Usage:       "mongo status [--uri <uri>]",
				Run:         svc.status,
			},
			{
				Name:        "ping",
				Description: "Envia ping ao servidor MongoDB",
				Usage:       "mongo ping [--uri <uri>]",
				Run:         svc.ping,
			},
			{
				Name:        "shell",
				Aliases:     []string{"connect", "cli"},
				Description: "Abre o shell interativo do MongoDB (mongosh)",
				Usage:       "mongo shell [--uri <uri>] [--db <banco>]",
				Run:         svc.shell,
			},
			{
				Name:        "dbs",
				Aliases:     []string{"databases", "list-dbs"},
				Description: "Lista os bancos de dados disponíveis",
				Usage:       "mongo dbs [--uri <uri>]",
				Run:         svc.dbs,
			},
			{
				Name:        "collections",
				Aliases:     []string{"cols", "ls"},
				Description: "Lista coleções de um banco de dados",
				Usage:       "mongo collections <banco> [--uri <uri>]",
				Run:         svc.collections,
			},
			{
				Name:        "count",
				Description: "Conta documentos em uma coleção",
				Usage:       "mongo count <banco> <colecao> [--uri <uri>]",
				Run:         svc.count,
			},
			{
				Name:        "stats",
				Description: "Mostra estatísticas de um banco de dados",
				Usage:       "mongo stats <banco> [--uri <uri>]",
				Run:         svc.stats,
			},
			{
				Name:        "indexes",
				Description: "Lista índices de uma coleção",
				Usage:       "mongo indexes <banco> <colecao> [--uri <uri>]",
				Run:         svc.indexes,
			},
			{
				Name:        "find",
				Description: "Busca documentos em uma coleção",
				Usage:       "mongo find <banco> <colecao> [--limit N] [--uri <uri>]",
				Run:         svc.find,
			},
			{
				Name:        "dump",
				Aliases:     []string{"export", "backup"},
				Description: "Exporta um banco de dados com mongodump",
				Usage:       "mongo dump <banco> [--out <diretorio>] [--uri <uri>]",
				Run:         svc.dump,
			},
			{
				Name:        "restore",
				Aliases:     []string{"import"},
				Description: "Restaura um banco de dados com mongorestore",
				Usage:       "mongo restore <banco> <diretorio> [--uri <uri>]",
				Run:         svc.restore,
			},
			{
				Name:        "drop-db",
				Description: "Remove um banco de dados (pede confirmação)",
				Usage:       "mongo drop-db <banco> [--uri <uri>] [-y]",
				Run:         svc.dropDB,
			},
			{
				Name:        "eval",
				Aliases:     []string{"run", "exec"},
				Description: "Executa um script JavaScript no MongoDB",
				Usage:       "mongo eval <banco> <script> [--uri <uri>]",
				Run:         svc.eval,
			},
			{
				Name:        "raw",
				Description: "Envia argumentos diretamente para mongosh/mongo",
				Usage:       "mongo raw <args...>",
				Run:         svc.raw,
			},
		},
		Examples: []string{
			"mongo status",
			"mongo dbs",
			"mongo collections mydb",
			"mongo find mydb users --limit 5",
			"mongo shell --uri mongodb://localhost:27017",
			"mongo dump mydb --out ./backup",
			"mongo eval mydb \"db.users.countDocuments()\"",
		},
	}
}

// resolveBin detecta o binário disponível: mongosh tem prioridade sobre mongo
func resolveBin(cfg config.Config) string {
	if b := os.Getenv("DUCK_MONGO_BIN"); b != "" {
		return b
	}
	if cfg.MongoBin != "" {
		return cfg.MongoBin
	}
	for _, candidate := range []string{"mongosh", "mongo"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}
	return "mongosh" // fallback; erro será exibido ao executar
}

// extractURI retira --uri <value> dos args e retorna uri + args restantes
func extractURI(args []string) (string, []string) {
	uri := ""
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--uri" && i+1 < len(args) {
			uri = args[i+1]
			i++
		} else {
			rest = append(rest, args[i])
		}
	}
	if uri == "" {
		uri = os.Getenv("DUCK_MONGO_URI")
	}
	return uri, rest
}

// extractFlag retira --flag <value> e retorna value + args restantes
func extractFlag(flag string, args []string) (string, []string) {
	val := ""
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == flag && i+1 < len(args) {
			val = args[i+1]
			i++
		} else {
			rest = append(rest, args[i])
		}
	}
	return val, rest
}

func (s service) mongoArgs(uri string, extra ...string) []string {
	args := []string{}
	if uri != "" {
		args = append(args, uri)
	}
	args = append(args, extra...)
	return args
}

func (s service) runInteractive(args ...string) error {
	return s.runner.Run(s.bin, args, runner.Options{
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Interactive: true,
	})
}

func (s service) runScript(uri, db, script string) error {
	args := []string{}
	if uri != "" {
		args = append(args, uri)
	}
	if db != "" {
		args = append(args, db)
	}
	args = append(args, "--eval", script, "--quiet")
	return s.runner.Run(s.bin, args, runner.Options{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}

func (s service) status(_ cli.Context, args []string) error {
	uri, _ := extractURI(args)
	fmt.Printf("Verificando MongoDB (%s)...\n", s.bin)
	if uri != "" {
		fmt.Printf("URI: %s\n", uri)
	}
	start := time.Now()
	script := `
		var info = db.runCommand({serverStatus: 1});
		print("Host:    " + info.host);
		print("Versao:  " + info.version);
		print("Uptime:  " + info.uptime + "s");
		print("Conex.:  " + info.connections.current + " ativas / " + info.connections.available + " disponíveis");
		print("Mem:     " + Math.round(info.mem.resident) + " MB resident");
	`
	baseDB := "admin"
	if err := s.runScript(uri, baseDB, script); err != nil {
		return fmt.Errorf("MongoDB inacessível: %w", err)
	}
	fmt.Printf("Latência: %dms\n", time.Since(start).Milliseconds())
	return nil
}

func (s service) ping(_ cli.Context, args []string) error {
	uri, _ := extractURI(args)
	fmt.Printf("Enviando ping ao MongoDB (%s)...\n", s.bin)
	start := time.Now()
	err := s.runScript(uri, "admin", `db.runCommand({ping:1}); print("pong");`)
	if err != nil {
		return fmt.Errorf("ping falhou: %w", err)
	}
	fmt.Printf("Resposta em %dms\n", time.Since(start).Milliseconds())
	return nil
}

func (s service) shell(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	dbName, rest := extractFlag("--db", rest)
	argv := []string{}
	if uri != "" {
		argv = append(argv, uri)
	}
	if dbName != "" {
		argv = append(argv, dbName)
	}
	argv = append(argv, rest...)
	return s.runInteractive(argv...)
}

func (s service) dbs(_ cli.Context, args []string) error {
	uri, _ := extractURI(args)
	return s.runScript(uri, "admin", `
		var dbs = db.adminCommand({listDatabases: 1});
		dbs.databases.forEach(function(d) {
			print(d.name + "\t" + Math.round(d.sizeOnDisk / 1024 / 1024) + " MB");
		});
	`)
}

func (s service) collections(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	if len(rest) == 0 {
		return cli.UsageError("informe o nome do banco: mongo collections <banco>")
	}
	dbName := rest[0]
	return s.runScript(uri, dbName, `
		db.getCollectionNames().forEach(function(c) {
			var stats = db.getCollection(c).stats();
			print(c + "\t" + stats.count + " docs\t" + Math.round(stats.size / 1024) + " KB");
		});
	`)
}

func (s service) count(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	if len(rest) < 2 {
		return cli.UsageError("use: mongo count <banco> <colecao>")
	}
	dbName, colName := rest[0], rest[1]
	script := fmt.Sprintf(`print(db.getCollection(%q).countDocuments());`, colName)
	return s.runScript(uri, dbName, script)
}

func (s service) stats(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	if len(rest) == 0 {
		return cli.UsageError("informe o banco: mongo stats <banco>")
	}
	dbName := rest[0]
	return s.runScript(uri, dbName, `
		var s = db.stats();
		print("Banco:       " + db.getName());
		print("Collections: " + s.collections);
		print("Documentos:  " + s.objects);
		print("Tamanho:     " + Math.round(s.dataSize / 1024 / 1024) + " MB");
		print("Storage:     " + Math.round(s.storageSize / 1024 / 1024) + " MB");
		print("Indices:     " + s.indexes);
	`)
}

func (s service) indexes(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	if len(rest) < 2 {
		return cli.UsageError("use: mongo indexes <banco> <colecao>")
	}
	dbName, colName := rest[0], rest[1]
	script := fmt.Sprintf(`
		db.getCollection(%q).getIndexes().forEach(function(idx) {
			print(JSON.stringify(idx.key) + "  name=" + idx.name);
		});
	`, colName)
	return s.runScript(uri, dbName, script)
}

func (s service) find(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	limitStr, rest := extractFlag("--limit", rest)
	if len(rest) < 2 {
		return cli.UsageError("use: mongo find <banco> <colecao> [--limit N]")
	}
	dbName, colName := rest[0], rest[1]
	limit := 10
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	script := fmt.Sprintf(`
		db.getCollection(%q).find().limit(%d).forEach(function(doc) {
			print(JSON.stringify(doc, null, 2));
		});
	`, colName, limit)
	return s.runScript(uri, dbName, script)
}

func (s service) dump(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	outDir, rest := extractFlag("--out", rest)
	if len(rest) == 0 {
		return cli.UsageError("informe o banco: mongo dump <banco> [--out <dir>]")
	}
	dbName := rest[0]
	if outDir == "" {
		outDir = fmt.Sprintf("./dump-%s-%s", dbName, time.Now().Format("20060102-150405"))
	}
	dumpArgs := []string{"--db", dbName, "--out", outDir}
	if uri != "" {
		dumpArgs = append([]string{"--uri", uri}, dumpArgs...)
	}
	fmt.Printf("Exportando banco '%s' para '%s'...\n", dbName, outDir)
	err := s.runner.Run("mongodump", dumpArgs, runner.Options{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return fmt.Errorf("mongodump falhou: %w", err)
	}
	fmt.Println("Export concluído.")
	return nil
}

func (s service) restore(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	if len(rest) < 2 {
		return cli.UsageError("use: mongo restore <banco> <diretorio>")
	}
	dbName, dir := rest[0], rest[1]
	restoreArgs := []string{"--db", dbName, dir}
	if uri != "" {
		restoreArgs = append([]string{"--uri", uri}, restoreArgs...)
	}
	fmt.Printf("Restaurando banco '%s' de '%s'...\n", dbName, dir)
	err := s.runner.Run("mongorestore", restoreArgs, runner.Options{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return fmt.Errorf("mongorestore falhou: %w", err)
	}
	fmt.Println("Restore concluído.")
	return nil
}

func (s service) dropDB(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	force := false
	filtered := rest[:0]
	for _, a := range rest {
		if a == "-y" || a == "--yes" {
			force = true
		} else {
			filtered = append(filtered, a)
		}
	}
	rest = filtered
	if len(rest) == 0 {
		return cli.UsageError("informe o banco: mongo drop-db <banco>")
	}
	dbName := rest[0]
	if !force {
		fmt.Printf("Tem certeza que deseja remover o banco '%s'? (s/N) ", dbName)
		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "s" && answer != "sim" && answer != "y" && answer != "yes" {
			fmt.Println("Operação cancelada.")
			return nil
		}
	}
	script := fmt.Sprintf(`db.getSiblingDB(%q).dropDatabase(); print("Banco removido.");`, dbName)
	return s.runScript(uri, "admin", script)
}

func (s service) eval(_ cli.Context, args []string) error {
	uri, rest := extractURI(args)
	if len(rest) < 2 {
		return cli.UsageError("use: mongo eval <banco> <script>")
	}
	dbName := rest[0]
	script := strings.Join(rest[1:], " ")
	return s.runScript(uri, dbName, script)
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para mongosh")
	}
	return s.runInteractive(args...)
}
