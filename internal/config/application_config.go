package config

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type ApplicationConfig struct {
	ServerAddress         string
	Database              DatabaseConfig
	BackgroundJobsEnabled bool
	Discovery             DiscoveryConfig
	Filtering             FilteringConfig
	Insight               InsightConfig
}

// InsightConfig holds the AI analyst's settings and the free material sources; the API key is read here and nowhere else.
type InsightConfig struct {
	AnthropicApiKey           string
	AnthropicBaseUrl          string
	Model                     string
	Effort                    string
	AnalysisTimeout           time.Duration
	MaximumConcurrentAnalyses int
	MaximumCoinsPerRound      int
	GoogleNewsBaseUrl         string
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

// FilteringConfig holds the filtering thresholds and where the filtering data sources live.
type FilteringConfig struct {
	MaximumTaxRate                  decimal.Decimal
	MinimumDailyVolumeUsd           decimal.Decimal
	MinimumFullyDilutedValuationUsd decimal.Decimal
	MaximumFullyDilutedValuationUsd decimal.Decimal
	MinimumCirculatingRatio         decimal.Decimal
	UnlockLookahead                 time.Duration
	MaximumUnlockRatioOfCirculating decimal.Decimal
	SourceRequestTimeout            time.Duration
	// RoundBaseBudget bounds gathering a round's data, before the allowance each security lookup adds.
	RoundBaseBudget time.Duration
	// TokenSecurityRequestInterval spaces token security lookups to stay inside the free request rate.
	TokenSecurityRequestInterval time.Duration
	GoPlusBaseUrl                string
	DefiLlamaDatasetsBaseUrl     string
}

// The per-source item limit and timeout are fixed by the discovery rules, not operator settings.
const (
	informationSourceItemLimit      = 50
	informationSourceRequestTimeout = 15 * time.Second
	// filteringSourceRequestTimeout is longer than discovery's: the coin list and contract lists are large answers.
	filteringSourceRequestTimeout = 20 * time.Second
	// filteringRoundBaseBudget is the agreed round time before security lookups: 60 seconds.
	filteringRoundBaseBudget = 60 * time.Second
	// tokenSecurityRequestInterval keeps token security lookups near the free tier's thirty per minute.
	tokenSecurityRequestInterval = 2 * time.Second
	// insightAnalysisTimeout bounds one question to the analyst.
	insightAnalysisTimeout = 120 * time.Second
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
		Insight: InsightConfig{
			AnthropicApiKey:           os.Getenv("ANTHROPIC_API_KEY"),
			AnthropicBaseUrl:          os.Getenv("ANTHROPIC_BASE_URL"),
			Model:                     cmp.Or(os.Getenv("INSIGHT_MODEL"), "claude-opus-5-5"),
			Effort:                    cmp.Or(os.Getenv("INSIGHT_EFFORT"), "low"),
			AnalysisTimeout:           insightAnalysisTimeout,
			MaximumConcurrentAnalyses: parsePositiveIntWithDefault(os.Getenv("INSIGHT_MAX_CONCURRENT_ANALYSES"), 3),
			MaximumCoinsPerRound:      parsePositiveIntWithDefault(os.Getenv("INSIGHT_MAX_COINS_PER_ROUND"), 20),
			GoogleNewsBaseUrl:         cmp.Or(os.Getenv("GOOGLE_NEWS_BASE_URL"), "https://news.google.com"),
		},
		Filtering: FilteringConfig{
			MaximumTaxRate:                  parsePositiveDecimalWithDefault(os.Getenv("FILTER_MAXIMUM_TAX_RATE"), "0.1"),
			MinimumDailyVolumeUsd:           parsePositiveDecimalWithDefault(os.Getenv("FILTER_MINIMUM_DAILY_VOLUME_USD"), "1000000"),
			MinimumFullyDilutedValuationUsd: parsePositiveDecimalWithDefault(os.Getenv("FILTER_MINIMUM_FDV_USD"), "10000000"),
			MaximumFullyDilutedValuationUsd: parsePositiveDecimalWithDefault(os.Getenv("FILTER_MAXIMUM_FDV_USD"), "1000000000"),
			MinimumCirculatingRatio:         parsePositiveDecimalWithDefault(os.Getenv("FILTER_MINIMUM_CIRCULATING_RATIO"), "0.2"),
			UnlockLookahead:                 time.Duration(parsePositiveIntWithDefault(os.Getenv("FILTER_UNLOCK_LOOKAHEAD_DAYS"), 14)) * 24 * time.Hour,
			MaximumUnlockRatioOfCirculating: parsePositiveDecimalWithDefault(os.Getenv("FILTER_MAXIMUM_UNLOCK_RATIO"), "0.05"),
			SourceRequestTimeout:            filteringSourceRequestTimeout,
			RoundBaseBudget:                 filteringRoundBaseBudget,
			TokenSecurityRequestInterval:    tokenSecurityRequestInterval,
			GoPlusBaseUrl:                   cmp.Or(os.Getenv("GOPLUS_BASE_URL"), "https://api.gopluslabs.io"),
			DefiLlamaDatasetsBaseUrl:        cmp.Or(os.Getenv("DEFILLAMA_DATASETS_BASE_URL"), "https://defillama-datasets.llama.fi"),
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

// parsePositiveDecimalWithDefault treats zero, negatives and typos alike as "use the default".
func parsePositiveDecimalWithDefault(rawValue string, defaultValue string) decimal.Decimal {
	parsedValue, parseError := decimal.NewFromString(strings.TrimSpace(rawValue))
	if parseError != nil || !parsedValue.IsPositive() {
		return decimal.RequireFromString(defaultValue)
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
