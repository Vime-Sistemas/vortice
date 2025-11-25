package config

import (
	"os"
	"strings"
	"bufio"
	"log"
	"path/filepath"
	"io"
)

// GetBackends retorna a lista de backends a partir da variável BACKEND_URLS (vírgula separada)
// ou retorna um fallback padrão se não definida.
func GetBackends() []string {
	if s := os.Getenv("BACKEND_URLS"); s != "" {
		parts := strings.Split(s, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	// Retornar lista vazia por padrão evita que o proxy aponte para si mesmo
	// (o que faria o health check marcar o backend como down).
	return []string{}
}

// GetAppPort retorna a porta da aplicação a partir da variável APP_PORT ou o fallback "8080".
func GetAppPort() string {
	if p := os.Getenv("APP_PORT"); p != "" {
		return p
	}
	return "8080"
}

// LoadEnv loads environment variables from .env.local or .env (in that order)
// It returns the path of the file loaded or empty string if none found.
func LoadEnv() string {
	// try .env.local then .env
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	candidates := []string{filepath.Join(cwd, ".env.local"), filepath.Join(cwd, ".env")}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if err := loadFileToEnv(p); err == nil {
				return p
			} else {
				log.Printf("warning: failed loading env file %s: %v", p, err)
			}
		}
	}
	return ""
}

func loadFileToEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// split at first '='
			if idx := strings.IndexByte(line, '='); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				// remove surrounding quotes if present
				if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
					val = val[1:len(val)-1]
				}
				os.Setenv(key, val)
			}
		}
		if err == io.EOF {
			break
		}
	}
	return nil
}
