package config

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type ApplicationConfig struct {
	ServerAddress         string
	Database              DatabaseConfig
	BackgroundJobsEnabled bool
	Discovery             DiscoveryConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SslMode  string
}

func (databaseConfig DatabaseConfig) DataSourceName() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		databaseConfig.Host,
		databaseConfig.Port,
		databaseConfig.User,
		databaseConfig.Password,
		databaseConfig.Database,
		databaseConfig.SslMode,
	)
}

// DiscoveryConfig holds the discovery rules and where each free information source lives.
type DiscoveryConfig struct {
	Window               time.Duration
	ExcludedCoinSymbols  []string
	ItemLimitPerSource   int
	SourceRequestTimeout time.Duration
	BinanceWebBaseUrl    string
	BinanceFuturesUrl    string
	BybitBaseUrl         string
	OkxBaseUrl           string
	CoinGeckoBaseUrl     string
	DexScreenerBaseUrl   string
}

// The per-source item limit and timeout are fixed by the discovery rules, not operator settings.
const (
	informationSourceItemLimit      = 50
	informationSourceRequestTimeout = 15 * time.Second
)

// defaultExcludedCoinSymbols are majors and stablecoins: never new coins, however often they are mentioned.
const defaultExcludedCoinSymbols = "BTC,ETH,BNB,SOL,XRP,USDT,USDC,FDUSD,DAI,TUSD,USDE"

// Load reads the process environment once at startup; every setting has a default so an empty .env still boots.
func Load() ApplicationConfig {
	return ApplicationConfig{
		ServerAddress: cmp.Or(os.Getenv("SERVER_ADDRESS"), ":8080"),
		Database: DatabaseConfig{
			Host:     cmp.Or(os.Getenv("POSTGRES_HOST"), "localhost"),
			Port:     cmp.Or(os.Getenv("POSTGRES_PORT"), "5432"),
			User:     cmp.Or(os.Getenv("POSTGRES_USER"), "postgres"),
			Password: cmp.Or(os.Getenv("POSTGRES_PASSWORD"), "postgres"),
			Database: cmp.Or(os.Getenv("POSTGRES_DATABASE"), "go_coin_hunter"),
			SslMode:  cmp.Or(os.Getenv("POSTGRES_SSL_MODE"), "disable"),
		},
		BackgroundJobsEnabled: parseBoolWithDefault(os.Getenv("BACKGROUND_JOBS_ENABLED"), true),
		Discovery: DiscoveryConfig{
			Window:               time.Duration(parsePositiveIntWithDefault(os.Getenv("DISCOVERY_WINDOW_HOURS"), 72)) * time.Hour,
			ExcludedCoinSymbols:  parseCommaSeparated(cmp.Or(os.Getenv("DISCOVERY_EXCLUDED_COIN_SYMBOLS"), defaultExcludedCoinSymbols)),
			ItemLimitPerSource:   informationSourceItemLimit,
			SourceRequestTimeout: informationSourceRequestTimeout,
			BinanceWebBaseUrl:    cmp.Or(os.Getenv("BINANCE_WEB_BASE_URL"), "https://www.binance.com"),
			BinanceFuturesUrl:    cmp.Or(os.Getenv("BINANCE_FUTURES_BASE_URL"), "https://fapi.binance.com"),
			BybitBaseUrl:         cmp.Or(os.Getenv("BYBIT_BASE_URL"), "https://api.bybit.com"),
			OkxBaseUrl:           cmp.Or(os.Getenv("OKX_BASE_URL"), "https://www.okx.com"),
			CoinGeckoBaseUrl:     cmp.Or(os.Getenv("COINGECKO_BASE_URL"), "https://api.coingecko.com"),
			DexScreenerBaseUrl:   cmp.Or(os.Getenv("DEXSCREENER_BASE_URL"), "https://api.dexscreener.com"),
		},
	}
}

func parseBoolWithDefault(rawValue string, defaultValue bool) bool {
	parsedValue, parseError := strconv.ParseBool(rawValue)
	if parseError != nil {
		return defaultValue
	}

	return parsedValue
}

// parsePositiveIntWithDefault treats zero, negatives and typos alike as "use the default".
func parsePositiveIntWithDefault(rawValue string, defaultValue int) int {
	parsedValue, parseError := strconv.Atoi(strings.TrimSpace(rawValue))
	if parseError != nil || parsedValue <= 0 {
		return defaultValue
	}

	return parsedValue
}

func parseCommaSeparated(rawValue string) []string {
	values := []string{}
	for _, value := range strings.Split(rawValue, ",") {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			values = append(values, trimmed)
		}
	}

	return values
}
