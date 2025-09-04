package utils

import (
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func TruncateText(s string, maxLength int) string {
	if utf8.RuneCountInString(s) <= maxLength {
		return s
	}

	if maxLength <= 3 {
		return "..."
	}

	runes := []rune(s)
	return string(runes[:maxLength-3]) + "..."
}

// formatDate formats a Unix timestamp string into the "Month Day, Year at Hour:Minute PM Timezone" layout.
func FormatDate(timestampStr string) string {
	t, err := ParseTimestamp(timestampStr)
	if err != nil {
		slog.Warn("Could not parse timestamp string", "timestampStr", timestampStr, "error", err)
		return timestampStr
	}

	// 1. Convert the time to the UTC timezone.
	t = t.UTC()

	// 2. Use the new format layout string.
	//    Based on Go's reference time: Mon Jan 2 15:04:05 MST 2006
	//    "January" -> Full month name (e.g., "September")
	//    "2"       -> Day of the month without leading zero (e.g., "2")
	//    "2006"    -> Four-digit year (e.g., "2025")
	//    "at"      -> The literal string "at"
	//    "3"       -> Hour in 12-hour format without leading zero (e.g., "4")
	//    "04"      -> Minute with leading zero (e.g., "04")
	//    "PM"      -> AM/PM marker (e.g., "PM")
	//    "MST"     -> Timezone abbreviation (will display "UTC" since we converted to UTC)
	return t.Format("January 2, 2006 at 3:04 PM MST")
}

// formatLargeNumber formats a number string into a human-readable string with k, M, B, T suffixes.
// It now accepts a string parameter for more flexibility.
func FormatLargeNumber(numStr string) string {
	// First, parse the string input into a float64.
	n, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		// If parsing fails, log a warning and return the original string.
		slog.Warn("Could not parse number string", "numStr", numStr, "error", err)
		return numStr
	}

	// Handle the zero case.
	if n == 0 {
		return "0"
	}

	// Handle negative numbers.
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}

	if n >= 1e12 { // Trillion
		return sign + FormatDecimal(n/1e12) + "T"
	} else if n >= 1e9 { // Billion
		return sign + FormatDecimal(n/1e9) + "B"
	} else if n >= 1e6 { // Million
		return sign + FormatDecimal(n/1e6) + "M"
	} else if n >= 1e3 { // Thousand
		return sign + FormatDecimal(n/1e3) + "k"
	}

	// For numbers less than 1000, return the formatted decimal string.
	return sign + FormatDecimal(n)
}

// formatDecimal is a helper function to format the number to at most 2 decimal places,
// removing trailing zeros and the decimal point if not needed.
func FormatDecimal(n float64) string {
	// Format to 2 decimal places initially.
	s := fmt.Sprintf("%.2f", n)
	// If it contains a decimal point, trim trailing zeros, then trim the trailing decimal point.
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func CalculateBigIntRatioPercentage(numerator string, denominator string) float64 {
	denominatorInt, ok := new(big.Int).SetString(denominator, 10)
	if !ok {
		slog.Warn("Invalid denominator string for ratio calculation", "value", denominator)
		return 0
	}

	if denominatorInt.Sign() == 0 {
		return 0
	}

	numeratorInt, ok := new(big.Int).SetString(numerator, 10)
	if !ok {
		slog.Warn("Invalid numerator string for ratio calculation", "value", numerator)
		numeratorInt = big.NewInt(0) // Invalid numerator, treat as zero.
	}

	numeratorFloat := new(big.Float).SetPrec(256).SetInt(numeratorInt)
	denominatorFloat := new(big.Float).SetPrec(256).SetInt(denominatorInt)

	ratio := new(big.Float).Quo(numeratorFloat, denominatorFloat)

	percent := new(big.Float).Mul(ratio, big.NewFloat(100))

	percentFloat64, _ := percent.Float64()
	return percentFloat64
}

func FormatPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value)
}

func FormatDurationShort(d time.Duration) string {
	// If the duration is in the past or now, the event has ended.
	if d <= 0 {
		return "Ended"
	}

	// Calculate individual units.
	days := d / (24 * time.Hour)
	d -= days * (24 * time.Hour)

	hours := d / time.Hour
	d -= hours * time.Hour

	minutes := d / time.Minute
	d -= minutes * time.Minute

	seconds := d / time.Second

	// Build a slice of the parts of the string.
	var parts []string

	if days > 0 {
		if days == 1 {
			parts = append(parts, "1 day")
		} else {
			parts = append(parts, fmt.Sprintf("%d days", days))
		}
	}

	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hour")
		} else {
			parts = append(parts, fmt.Sprintf("%d hours", hours))
		}
	}

	if minutes > 0 && len(parts) < 2 { // Only add if we have less than 2 parts
		if minutes == 1 {
			parts = append(parts, "1 minute")
		} else {
			parts = append(parts, fmt.Sprintf("%d minutes", minutes))
		}
	}

	if seconds > 0 && len(parts) < 2 { // Only add if we have less than 2 parts
		if seconds == 1 {
			parts = append(parts, "1 second")
		} else {
			parts = append(parts, fmt.Sprintf("%d seconds", seconds))
		}
	}

	// If there are no parts (duration is < 1s), return a default.
	if len(parts) == 0 {
		return "< 1 second"
	}

	return strings.Join(parts, ", ")
}

func FormatBigIntWithDecimals(amountStr *string, decimals int) (string, error) {
	if amountStr == nil {
		return "", nil
	}

	if *amountStr == "" {
		return "", nil
	}

	amountInt, ok := new(big.Int).SetString(*amountStr, 10)
	if !ok {
		return "", fmt.Errorf("invalid bigint string: %s", *amountStr)
	}

	if decimals < 0 {
		return "", fmt.Errorf("decimals cannot be negative: %d", decimals)
	}
	if decimals == 0 || decimals == 1 {
		return *amountStr, nil
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)

	amountFloat := new(big.Float).SetPrec(256).SetInt(amountInt)
	divisorFloat := new(big.Float).SetPrec(256).SetInt(divisor)

	resultFloat := new(big.Float).Quo(amountFloat, divisorFloat)

	formattedStr := resultFloat.Text('f', decimals)

	if strings.Contains(formattedStr, ".") {
		formattedStr = strings.TrimRight(formattedStr, "0")
		formattedStr = strings.TrimRight(formattedStr, ".")
	}

	return formattedStr, nil
}

// FormatAsQuote takes a multi-line string and prefixes each line with "> ".
func FormatAsMdQuote(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")

	var quotedLines []string

	for _, line := range lines {
		quotedLines = append(quotedLines, "> "+line)
	}

	return strings.Join(quotedLines, "\n")
}
