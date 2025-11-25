package config

import (
	"os"
	"strings"
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
	return []string{"http://localhost:8080"}
}

// GetAppPort retorna a porta da aplicação a partir da variável APP_PORT ou o fallback "8080".
func GetAppPort() string {
	if p := os.Getenv("APP_PORT"); p != "" {
		return p
	}
	return "8080"
}
