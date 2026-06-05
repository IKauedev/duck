package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	"gopkg.in/yaml.v3"

	"duck/internal/cli"
)

const encryptedPrefix = "DUCK-ENC-v1"

func EncryptCommand() cli.Command {
	return cli.Command{Name: "encrypt", Description: "Criptografa arquivo local com senha", Usage: "encrypt <arquivo> [saida] [--pass senha]", Run: encryptFile}
}

func DecryptCommand() cli.Command {
	return cli.Command{Name: "decrypt", Description: "Decriptografa arquivo local com senha", Usage: "decrypt <arquivo> [saida] [--pass senha]", Run: decryptFile}
}

func PasswordCommand() cli.Command {
	return cli.Command{Name: "password", Description: "Gera senhas, tokens e secrets seguros", Usage: "password [--length N] [--token bytes]", Run: password}
}

func QRCommand() cli.Command {
	return cli.Command{Name: "qr", Description: "Gera QR Code no terminal", Usage: "qr <texto|url>", Run: qr}
}

func ServeCommand() cli.Command {
	return cli.Command{Name: "serve", Description: "Serve uma pasta via HTTP local", Usage: "serve [pasta] [--port porta] [--host host]", Run: serve}
}

func CIDRCommand() cli.Command {
	return cli.Command{
		Name:        "cidr",
		Description: "Calcula redes, overlaps e IPs AWS utilizaveis",
		Usage:       "cidr <calc|aws|overlap>",
		Children: []cli.Command{
			{Name: "calc", Description: "Mostra informacoes de um CIDR", Usage: "cidr calc <cidr>", Run: cidrCalc},
			{Name: "aws", Description: "Calcula IPs utilizaveis em subnet AWS", Usage: "cidr aws <cidr>", Run: cidrAWS},
			{Name: "overlap", Description: "Verifica overlap entre CIDRs", Usage: "cidr overlap <cidr1> <cidr2>", Run: cidrOverlap},
		},
	}
}

func CalcCommand() cli.Command {
	return cli.Command{
		Name:        "calc",
		Description: "Calculadoras rapidas",
		Usage:       "calc <ip>",
		Children: []cli.Command{
			{Name: "ip", Description: "Alias para cidr aws", Usage: "calc ip <cidr>", Run: cidrAWS},
		},
	}
}

func JSONCommand() cli.Command {
	return cli.Command{Name: "json", Description: "Formata, valida e consulta JSON", Usage: "json <format|validate|get>", Run: jsonCommand}
}

func YAMLCommand() cli.Command {
	return cli.Command{Name: "yaml", Description: "Formata e valida YAML", Usage: "yaml <format|validate>", Run: yamlCommand}
}

func encryptFile(_ cli.Context, args []string) error {
	opts, err := parseCryptoArgs(args)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(filepath.Clean(opts.input))
	if err != nil {
		return err
	}
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	key := deriveKey(opts.password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	ciphertext := gcm.Seal(nil, nonce, content, nil)
	payload := strings.Join([]string{
		encryptedPrefix,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(ciphertext),
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Clean(opts.output), []byte(payload), 0600); err != nil {
		return err
	}
	fmt.Println("Arquivo criptografado:", opts.output)
	return nil
}

func decryptFile(_ cli.Context, args []string) error {
	opts, err := parseCryptoArgs(args)
	if err != nil {
		return err
	}
	if opts.output == opts.input+".duck" && strings.HasSuffix(opts.input, ".duck") {
		opts.output = strings.TrimSuffix(opts.input, ".duck")
	}
	content, err := os.ReadFile(filepath.Clean(opts.input))
	if err != nil {
		return err
	}
	parts := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(parts) != 4 || parts[0] != encryptedPrefix {
		return fmt.Errorf("arquivo nao parece ter sido criptografado pelo Duck")
	}
	salt, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	nonce, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return err
	}
	key := deriveKey(opts.password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("senha invalida ou arquivo corrompido")
	}
	if err := os.WriteFile(filepath.Clean(opts.output), plaintext, 0600); err != nil {
		return err
	}
	fmt.Println("Arquivo decriptografado:", opts.output)
	return nil
}

type cryptoOptions struct {
	input    string
	output   string
	password string
}

func parseCryptoArgs(args []string) (cryptoOptions, error) {
	opts := cryptoOptions{}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--pass":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--pass precisa de um valor")
			}
			opts.password = args[i+1]
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) < 1 || len(rest) > 2 {
		return opts, cli.UsageError("use: encrypt/decrypt <arquivo> [saida] [--pass senha]")
	}
	if opts.password == "" {
		return opts, cli.UsageError("informe --pass senha")
	}
	opts.input = rest[0]
	if len(rest) == 2 {
		opts.output = rest[1]
	} else {
		opts.output = rest[0] + ".duck"
	}
	return opts, nil
}

func deriveKey(password string, salt []byte) []byte {
	return pbkdf2SHA256([]byte(password), salt, 200000, 32)
}

func pbkdf2SHA256(password, salt []byte, iterations int, keyLength int) []byte {
	hashLength := sha256.Size
	blocks := (keyLength + hashLength - 1) / hashLength
	key := make([]byte, 0, blocks*hashLength)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		var index [4]byte
		binary.BigEndian.PutUint32(index[:], uint32(block))
		mac.Write(index[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		key = append(key, t...)
	}
	return key[:keyLength]
}

func password(_ cli.Context, args []string) error {
	length := 32
	tokenBytes := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--length", "-l":
			if i+1 >= len(args) {
				return cli.UsageError(args[i] + " precisa de um valor")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil || value <= 0 {
				return cli.UsageError(args[i] + " precisa ser numero positivo")
			}
			length = value
			i++
		case "--token":
			if i+1 >= len(args) {
				return cli.UsageError("--token precisa de um valor")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil || value <= 0 {
				return cli.UsageError("--token precisa ser numero positivo")
			}
			tokenBytes = value
			i++
		default:
			return cli.UsageError("opcao invalida para password: " + args[i])
		}
	}
	if tokenBytes > 0 {
		raw := make([]byte, tokenBytes)
		if _, err := rand.Read(raw); err != nil {
			return err
		}
		fmt.Println(hex.EncodeToString(raw))
		return nil
	}
	value, err := randomPassword(length)
	if err != nil {
		return err
	}
	fmt.Println(value)
	return nil
}

func randomPassword(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}"
	var out strings.Builder
	max := big.NewInt(int64(len(alphabet)))
	for out.Len() < length {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out.WriteByte(alphabet[index.Int64()])
	}
	return out.String(), nil
}

func qr(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: qr <texto|url>")
	}
	code, err := qrcode.New(args[0], qrcode.Medium)
	if err != nil {
		return err
	}
	fmt.Print(code.ToSmallString(false))
	return nil
}

func serve(_ cli.Context, args []string) error {
	dir := "."
	host := "127.0.0.1"
	port := "8080"
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 >= len(args) {
				return cli.UsageError(args[i] + " precisa de um valor")
			}
			port = args[i+1]
			i++
		case "--host":
			if i+1 >= len(args) {
				return cli.UsageError("--host precisa de um valor")
			}
			host = args[i+1]
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) > 1 {
		return cli.UsageError("use: serve [pasta] [--port porta] [--host host]")
	}
	if len(rest) == 1 {
		dir = rest[0]
	}
	address := net.JoinHostPort(host, port)
	fmt.Println("Servindo", dir, "em http://"+address)
	return http.ListenAndServe(address, http.FileServer(http.Dir(dir)))
}

func jsonCommand(_ cli.Context, args []string) error {
	if len(args) < 1 {
		return cli.UsageError("use: json <format|validate|get> [arquivo] [path]")
	}
	switch args[0] {
	case "format":
		content, err := readArgFile(args[1:])
		if err != nil {
			return err
		}
		var value any
		if err := json.Unmarshal(content, &value); err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
	case "validate":
		content, err := readArgFile(args[1:])
		if err != nil {
			return err
		}
		if !json.Valid(content) {
			return fmt.Errorf("JSON invalido")
		}
		fmt.Println("JSON valido.")
	case "get":
		if len(args) != 3 {
			return cli.UsageError("use: json get <arquivo> <path>")
		}
		content, err := os.ReadFile(filepath.Clean(args[1]))
		if err != nil {
			return err
		}
		var value any
		if err := json.Unmarshal(content, &value); err != nil {
			return err
		}
		result, ok := jsonPath(value, args[2])
		if !ok {
			return fmt.Errorf("path nao encontrado: %s", args[2])
		}
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
	default:
		return cli.UsageError("subcomando invalido para json: " + args[0])
	}
	return nil
}

func yamlCommand(_ cli.Context, args []string) error {
	if len(args) < 1 {
		return cli.UsageError("use: yaml <format|validate> [arquivo]")
	}
	content, err := readArgFile(args[1:])
	if err != nil {
		return err
	}
	var value any
	if err := yaml.Unmarshal(content, &value); err != nil {
		return err
	}
	switch args[0] {
	case "format":
		encoded, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		fmt.Print(string(encoded))
	case "validate":
		fmt.Println("YAML valido.")
	default:
		return cli.UsageError("subcomando invalido para yaml: " + args[0])
	}
	return nil
}

func readArgFile(args []string) ([]byte, error) {
	if len(args) == 0 {
		return io.ReadAll(os.Stdin)
	}
	if len(args) != 1 {
		return nil, cli.UsageError("informe no maximo um arquivo")
	}
	return os.ReadFile(filepath.Clean(args[0]))
}

func jsonPath(value any, path string) (any, bool) {
	current := value
	for _, part := range strings.Split(strings.TrimPrefix(path, "."), ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func cidrCalc(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: cidr calc <cidr>")
	}
	info, err := cidrInfo(args[0])
	if err != nil {
		return err
	}
	printCIDRInfo(info, false)
	return nil
}

func cidrAWS(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: cidr aws <cidr>")
	}
	info, err := cidrInfo(args[0])
	if err != nil {
		return err
	}
	printCIDRInfo(info, true)
	return nil
}

func cidrOverlap(_ cli.Context, args []string) error {
	if len(args) != 2 {
		return cli.UsageError("use: cidr overlap <cidr1> <cidr2>")
	}
	first, err := cidrInfo(args[0])
	if err != nil {
		return err
	}
	second, err := cidrInfo(args[1])
	if err != nil {
		return err
	}
	overlap := first.network <= second.broadcast && second.network <= first.broadcast
	fmt.Println("Overlap:", overlap)
	return nil
}

type cidrDetails struct {
	ip        net.IP
	network   uint32
	broadcast uint32
	ones      int
	bits      int
	total     uint64
}

func cidrInfo(raw string) (cidrDetails, error) {
	ip, network, err := net.ParseCIDR(raw)
	if err != nil {
		return cidrDetails{}, err
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return cidrDetails{}, fmt.Errorf("apenas IPv4 e suportado por enquanto")
	}
	ones, bits := network.Mask.Size()
	networkIP := binary.BigEndian.Uint32(network.IP.To4())
	hostCount := uint64(1) << uint(bits-ones)
	return cidrDetails{
		ip:        ip4,
		network:   networkIP,
		broadcast: networkIP + uint32(hostCount) - 1,
		ones:      ones,
		bits:      bits,
		total:     hostCount,
	}, nil
}

func printCIDRInfo(info cidrDetails, aws bool) {
	fmt.Println("CIDR:", uint32ToIP(info.network).String()+"/"+strconv.Itoa(info.ones))
	fmt.Println("Tipo:", publicPrivate(info.ip))
	fmt.Println("Network:", uint32ToIP(info.network))
	fmt.Println("Broadcast:", uint32ToIP(info.broadcast))
	fmt.Println("Total IPs:", info.total)
	if aws {
		if info.ones < 16 || info.ones > 28 {
			fmt.Println("Aviso AWS: subnets VPC normalmente precisam estar entre /16 e /28.")
		}
		usable := uint64(0)
		if info.total > 5 {
			usable = info.total - 5
		}
		fmt.Println("AWS utilizaveis:", usable)
		fmt.Println("AWS reservados:")
		for offset := uint32(0); offset < 4 && uint64(offset) < info.total; offset++ {
			fmt.Println(" ", uint32ToIP(info.network+offset))
		}
		if info.total > 1 {
			fmt.Println(" ", uint32ToIP(info.broadcast))
		}
		if info.total > 5 {
			fmt.Println("Primeiro utilizavel AWS:", uint32ToIP(info.network+4))
			fmt.Println("Ultimo utilizavel AWS:", uint32ToIP(info.broadcast-1))
		}
	} else if info.total > 2 {
		fmt.Println("Hosts utilizaveis:", info.total-2)
		fmt.Println("Primeiro host:", uint32ToIP(info.network+1))
		fmt.Println("Ultimo host:", uint32ToIP(info.broadcast-1))
	}
}

func publicPrivate(ip net.IP) string {
	if ip.IsPrivate() {
		return "privado"
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return "especial"
	}
	return "publico"
}

func uint32ToIP(value uint32) net.IP {
	var buffer bytes.Buffer
	_ = binary.Write(&buffer, binary.BigEndian, value)
	return net.IP(buffer.Bytes())
}
