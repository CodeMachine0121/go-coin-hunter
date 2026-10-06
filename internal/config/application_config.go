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
	Verdict               VerdictConfig
	MarketStructure       MarketStructureConfig
	// HuntPipelineInterval is how often the scheduled hunt round runs; zero or less switches the schedule off.
	HuntPipelineInterval time.Duration
	// ShutdownGracePeriod is how long shutdown waits for the step in flight before abandoning it; an abandoned step's
	// run is marked interrupted at the next start.
	ShutdownGracePeriod time.Duration
}

// VerdictConfig holds the chief investment officer's model settings; the API key is shared with the insight step.
type VerdictConfig struct {
	Model            string
	Effort           string
	SynthesisTimeout time.Duration
	// MinimumBullishInsightStrength is the weakest bullish insight still handed to the chief investment officer.
	MinimumBullishInsightStrength int
	// MinimumHuntBoardConfidence is the least confidence a long verdict needs to be put on the hunt board.
	MinimumHuntBoardConfidence int
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
	// The material limits are fixed rules of the insight step, not operator settings.
	MaximumIntelligenceHeadlines int
	MaximumNewsHeadlines         int
	NewsLookback                 time.Duration
	MaterialSourceTimeout        time.Duration
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

// MarketStructureConfig is how every step asks the exchanges about a perpetual: one timeout per exchange and a cap on
// lookups at once, so a batch of coins stays inside the free request rates. Both are fixed rules, not operator settings.
type MarketStructureConfig struct {
	RequestTimeout           time.Duration
	MaximumConcurrentLookups int
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
	// The momentum thresholds are ratios as the exchanges report them: -0.1 is a 10% fall, 0.001 a 0.1% funding rate.
	MinimumPriceChangeRatio        decimal.Decimal
	MaximumPriceChangeRatio        decimal.Decimal
	MinimumOpenInterestChangeRatio decimal.Decimal
	MaximumFundingRate             decimal.Decimal
	// MarketStructureBudget bounds looking up every candidate's perpetual in one filtering round.
	MarketStructureBudget time.Duration
	SourceRequestTimeout  time.Duration
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
	// filteringMarketStructureBudget is how long one filtering round may spend asking the exchanges about its perpetuals.
	filteringMarketStructureBudget = 60 * time.Second
	// marketStructureRequestTimeout bounds asking one exchange about one coin's perpetual, a small answer.
	marketStructureRequestTimeout = 15 * time.Second
	// marketStructureMaximumConcurrentLookups keeps a batch of coins well inside the exchanges' free request rates.
	marketStructureMaximumConcurrentLookups = 5
	// filteringRoundBaseBudget is the agreed round time before security lookups: 60 seconds.
	filteringRoundBaseBudget = 60 * time.Second
	// tokenSecurityRequestInterval keeps token security lookups near the free tier's thirty per minute.
	tokenSecurityRequestInterval = 2 * time.Second
	// insightAnalysisTimeout bounds one question to the analyst.
	insightAnalysisTimeout = 120 * time.Second
	// verdictSynthesisTimeout is longer: the strategist weighs every coin of the round in one answer.
	verdictSynthesisTimeout = 180 * time.Second
)

// defaultExcludedCoinSymbols are majors and stablecoins: never new coins, however often they are mentioned.
const defaultExcludedCoinSymbols = "BTC,ETH,BNB,SOL,XRP,USDT,USDC,FDUSD,DAI,TUSD,USDE"

// Load reads the process environment once at startup; every setting has a default so an empty .env still boots.
func Load() ApplicationConfig {
	minimumPriceChangeRatio, maximumPriceChangeRatio := parseDecimalRangeWithDefault(
		os.Getenv("FILTER_MINIMUM_PRICE_CHANGE_RATIO"), os.Getenv("FILTER_MAXIMUM_PRICE_CHANGE_RATIO"), "-0.1", "0.6")
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
			AnthropicApiKey:              os.Getenv("ANTHROPIC_API_KEY"),
			AnthropicBaseUrl:             os.Getenv("ANTHROPIC_BASE_URL"),
			Model:                        cmp.Or(os.Getenv("INSIGHT_MODEL"), "claude-opus-5-5"),
			Effort:                       cmp.Or(os.Getenv("INSIGHT_EFFORT"), "low"),
			AnalysisTimeout:              insightAnalysisTimeout,
			MaximumConcurrentAnalyses:    parsePositiveIntWithDefault(os.Getenv("INSIGHT_MAX_CONCURRENT_ANALYSES"), 3),
			MaximumCoinsPerRound:         parsePositiveIntWithDefault(os.Getenv("INSIGHT_MAX_COINS_PER_ROUND"), 20),
			GoogleNewsBaseUrl:            cmp.Or(os.Getenv("GOOGLE_NEWS_BASE_URL"), "https://news.google.com"),
			MaximumIntelligenceHeadlines: 20,
			MaximumNewsHeadlines:         10,
			NewsLookback:                 72 * time.Hour,
			MaterialSourceTimeout:        15 * time.Second,
		},
		MarketStructure: MarketStructureConfig{
			RequestTimeout:           marketStructureRequestTimeout,
			MaximumConcurrentLookups: marketStructureMaximumConcurrentLookups,
		},
		HuntPipelineInterval: time.Duration(parseIntWithDefault(os.Getenv("HUNT_PIPELINE_INTERVAL_HOURS"), 4)) * time.Hour,
		ShutdownGracePeriod:  time.Duration(parsePositiveIntWithDefault(os.Getenv("SHUTDOWN_GRACE_MINUTES"), 15)) * time.Minute,
		Verdict: VerdictConfig{
			Model:                         cmp.Or(os.Getenv("VERDICT_MODEL"), "claude-opus-5-5"),
			Effort:                        cmp.Or(os.Getenv("VERDICT_EFFORT"), "high"),
			SynthesisTimeout:              verdictSynthesisTimeout,
			MinimumBullishInsightStrength: parsePositiveIntWithDefault(os.Getenv("VERDICT_MINIMUM_BULLISH_INSIGHT_STRENGTH"), 6),
			MinimumHuntBoardConfidence:    parseIntWithDefault(os.Getenv("HUNT_BOARD_MINIMUM_CONFIDENCE"), 50),
		},
		Filtering: FilteringConfig{
			MaximumTaxRate:                  parsePositiveDecimalWithDefault(os.Getenv("FILTER_MAXIMUM_TAX_RATE"), "0.1"),
			MinimumDailyVolumeUsd:           parsePositiveDecimalWithDefault(os.Getenv("FILTER_MINIMUM_DAILY_VOLUME_USD"), "1000000"),
			MinimumFullyDilutedValuationUsd: parsePositiveDecimalWithDefault(os.Getenv("FILTER_MINIMUM_FDV_USD"), "10000000"),
			MaximumFullyDilutedValuationUsd: parsePositiveDecimalWithDefault(os.Getenv("FILTER_MAXIMUM_FDV_USD"), "1000000000"),
			MinimumCirculatingRatio:         parsePositiveDecimalWithDefault(os.Getenv("FILTER_MINIMUM_CIRCULATING_RATIO"), "0.2"),
			UnlockLookahead:                 time.Duration(parsePositiveIntWithDefault(os.Getenv("FILTER_UNLOCK_LOOKAHEAD_DAYS"), 14)) * 24 * time.Hour,
			MaximumUnlockRatioOfCirculating: parsePositiveDecimalWithDefault(os.Getenv("FILTER_MAXIMUM_UNLOCK_RATIO"), "0.05"),
			MinimumPriceChangeRatio:         minimumPriceChangeRatio,
			MaximumPriceChangeRatio:         maximumPriceChangeRatio,
			MinimumOpenInterestChangeRatio:  parseDecimalWithDefault(os.Getenv("FILTER_MINIMUM_OPEN_INTEREST_CHANGE_RATIO"), "-0.1"),
			MaximumFundingRate:              parseDecimalWithDefault(os.Getenv("FILTER_MAXIMUM_FUNDING_RATE"), "0.001"),
			MarketStructureBudget:           filteringMarketStructureBudget,
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

// parseIntWithDefault falls back only on a typo or an empty value; zero and negatives are kept, as switches.
func parseIntWithDefault(rawValue string, defaultValue int) int {
	parsedValue, parseError := strconv.Atoi(strings.TrimSpace(rawValue))
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

// parseDecimalWithDefault falls back only on a typo or an empty value; zero and negatives are kept, as thresholds.
func parseDecimalWithDefault(rawValue string, defaultValue string) decimal.Decimal {
	parsedValue, parseError := decimal.NewFromString(strings.TrimSpace(rawValue))
	if parseError != nil {
		return decimal.RequireFromString(defaultValue)
	}

	return parsedValue
}

// parseDecimalRangeWithDefault reads a floor and a ceiling; a floor above its ceiling would reject everything, so the
// pair falls back to the defaults together.
func parseDecimalRangeWithDefault(rawMinimum string, rawMaximum string, defaultMinimum string, defaultMaximum string) (decimal.Decimal, decimal.Decimal) {
	minimum, maximum := parseDecimalWithDefault(rawMinimum, defaultMinimum), parseDecimalWithDefault(rawMaximum, defaultMaximum)
	if minimum.GreaterThan(maximum) {
		return decimal.RequireFromString(defaultMinimum), decimal.RequireFromString(defaultMaximum)
	}

	return minimum, maximum
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
