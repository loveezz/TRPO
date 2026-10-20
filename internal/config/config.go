package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	DatabaseDSN    string
	SessionSecret  string
	SeedCandidates int // сколько всего кандидатов сгенерировать при запуске (0 — не генерировать)
}

func Load() Config {
	cfg := Config{
		Port:          env("APP_PORT", "8000"),
		DatabaseDSN:   env("DATABASE_DSN", "host=localhost user=candidate_user password=candidate_pass dbname=candidates_db port=5432 sslmode=disable TimeZone=Asia/Yekaterinburg"),
		SessionSecret: env("SESSION_SECRET", "dev-secret-change-me"),
	}
	// Защитные проверки входных данных конфигурации с печатью переменных.
	if p, err := strconv.Atoi(strings.TrimSpace(cfg.Port)); err != nil || p < 1 || p > 65535 {
		log.Printf("[WARN] config: APP_PORT=%q некорректен, использован порт 8000", cfg.Port)
		cfg.Port = "8000"
	}
	if raw := strings.TrimSpace(os.Getenv("SEED_CANDIDATES")); raw != "" {
		if n, err := strconv.Atoi(raw); err != nil || n < 0 || n > 100000 {
			log.Printf("[WARN] config: SEED_CANDIDATES=%q некорректен (ожидается 0..100000), генерация отключена", raw)
		} else {
			cfg.SeedCandidates = n
		}
	}
	if strings.TrimSpace(cfg.DatabaseDSN) == "" {
		log.Printf("[WARN] config: DATABASE_DSN пуст")
	}
	if cfg.SessionSecret == "dev-secret-change-me" {
		log.Printf("[WARN] config: используется SESSION_SECRET по умолчанию, замените в production")
	}
	if os.Getenv("DATABASE_DSN") == "" {
		log.Printf("[WARN] config: переменная DATABASE_DSN не задана, использовано значение по умолчанию")
	}
	return cfg
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
