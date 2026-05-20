package output

import (
	"fmt"
	"io"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var printer = message.NewPrinter(language.English)

// formatUSD formats a USDT amount with thousands separators and 2 decimals.
func formatUSD(v float64) string {
	return printer.Sprintf("$%.2f", v)
}

// WritePortfolio writes the portfolio summary in human-readable form.
func WritePortfolio(w io.Writer, balance, available, upnl float64, positions int) {
	fmt.Fprintf(w, "Balance:           %s USDT\n", formatUSD(balance))
	fmt.Fprintf(w, "Available margin:  %s USDT\n", formatUSD(available))
	fmt.Fprintf(w, "Positions:         %d  (uPnL: %s)\n", positions, formatUSD(upnl))
}

// PrintLine writes parts separated by two spaces, terminated by a newline.
// Used by positions/orders commands to render simple table rows.
func PrintLine(w io.Writer, parts ...any) {
	for i, p := range parts {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprint(w, p)
	}
	fmt.Fprintln(w)
}
