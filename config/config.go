package config

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
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
					val = val[1 : len(val)-1]
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

// StartLocalBackendsEnabled retorna true se a variável START_LOCAL_BACKENDS estiver definida como "true" (case-insensitive).
func StartLocalBackendsEnabled() bool {
	v := os.Getenv("START_LOCAL_BACKENDS")
	return strings.EqualFold(v, "true")
}

// GetLocalBackendStartPort retorna a porta inicial para backends locais (LOCAL_BACKEND_START_PORT), fallback 8081.
func GetLocalBackendStartPort() int {
	if s := os.Getenv("LOCAL_BACKEND_START_PORT"); s != "" {
		if p, err := strconv.Atoi(s); err == nil {
			return p
		}
	}
	return 8081
}

// GetLocalBackendCount retorna quantos backends locais iniciar (LOCAL_BACKEND_COUNT), fallback 3.
func GetLocalBackendCount() int {
	if s := os.Getenv("LOCAL_BACKEND_COUNT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return 3
}

// GetLocalBackendForce retorna true se LOCAL_BACKEND_FORCE estiver definida como "true".
// Quando true, os backends locais substituem qualquer BACKEND_URLS configurado.
func GetLocalBackendForce() bool {
	v := os.Getenv("LOCAL_BACKEND_FORCE")
	return strings.EqualFold(v, "true")
}

// GetLBAlgorithm retorna o algoritmo de balanceamento. Valores suportados: "round_robin", "least_conn", "random", "ip_hash".
func GetLBAlgorithm() string {
	if s := os.Getenv("LOAD_BALANCER_ALGO"); s != "" {
		return strings.ToLower(s)
	}
	return "round_robin"
}

// GetRateLimitRPS retorna o rate limit em requisições por segundo (0 = disabled)
func GetRateLimitRPS() int {
	if s := os.Getenv("RATE_LIMIT_RPS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return 0
}

// GetRateLimitBurst retorna o burst para o rate limiter (default 1)
func GetRateLimitBurst() int {
	if s := os.Getenv("RATE_LIMIT_BURST"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return 1
}

// GetIPHashHeader retorna o nome do header a ser usado para ip_hash (ex: "X-Forwarded-For").
// Se vazio, usa RemoteAddr.
func GetIPHashHeader() string {
	return os.Getenv("IP_HASH_HEADER")
}

// GetPerBackendRateLimits parses BACKEND_RATE_LIMITS no formato "rps/burst,rps/burst,...".
// Retorna um slice de pares [][2]int com (rps, burst) para cada backend; se fewer entries,
// preenche com os valores globais de RATE_LIMIT_RPS e RATE_LIMIT_BURST.
func GetPerBackendRateLimits(n int) [][2]int {
	out := make([][2]int, 0, n)
	globalRPS := GetRateLimitRPS()
	globalBurst := GetRateLimitBurst()
	s := os.Getenv("BACKEND_RATE_LIMITS")
	if s == "" {
		for i := 0; i < n; i++ {
			out = append(out, [2]int{globalRPS, globalBurst})
		}
		return out
	}
	parts := strings.Split(s, ",")
	for i := 0; i < n; i++ {
		if i < len(parts) {
			p := strings.TrimSpace(parts[i])
			if p == "" {
				out = append(out, [2]int{globalRPS, globalBurst})
				continue
			}
			// parse rps/burst
			if idx := strings.IndexByte(p, '/'); idx > 0 {
				rpsStr := strings.TrimSpace(p[:idx])
				burstStr := strings.TrimSpace(p[idx+1:])
				rps, err1 := strconv.Atoi(rpsStr)
				burst, err2 := strconv.Atoi(burstStr)
				if err1 == nil && err2 == nil {
					out = append(out, [2]int{rps, burst})
					continue
				}
			}
			// fallback: try parse single number as rps
			if rps, err := strconv.Atoi(p); err == nil {
				out = append(out, [2]int{rps, globalBurst})
				continue
			}
			// invalid, use global defaults
			out = append(out, [2]int{globalRPS, globalBurst})
		} else {
			out = append(out, [2]int{globalRPS, globalBurst})
		}
	}
	return out
}
