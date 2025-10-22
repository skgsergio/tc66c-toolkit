package main

import (
	"fmt"
	"os"
	"time"

	"github.com/skgsergio/tc66-toolkit/lib/tc66c"
	"github.com/spf13/cobra"
)

var (
	intervalFlag time.Duration
	pollJSONFlag bool
)

var pollCmd = &cobra.Command{
	Use:   "poll",
	Short: "Continuously poll readings from the device",
	Run: func(cmd *cobra.Command, args []string) {
		device := connectDevice(portFlag)
		defer device.Close()
		executePoll(device, intervalFlag, pollJSONFlag)
	},
}

func init() {
	pollCmd.Flags().DurationVarP(&intervalFlag, "interval", "i", 500*time.Millisecond, "Polling interval")
	pollCmd.Flags().BoolVarP(&pollJSONFlag, "json", "j", false, "Output in JSON format")
	rootCmd.AddCommand(pollCmd)
}

// executePoll continuously polls readings from the device
func executePoll(tc66c *tc66c.TC66C, interval time.Duration, jsonOutput bool) {
	if !jsonOutput {
		fmt.Printf("Polling readings every %v (press Ctrl+C to stop)...\n\n", interval)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Get first reading immediately
	printReading(tc66c, jsonOutput)

	// Poll at specified interval
	for range ticker.C {
		printReading(tc66c, jsonOutput)
	}
}

// printReading gets and prints a single reading
func printReading(tc66c *tc66c.TC66C, jsonOutput bool) {
	reading, err := tc66c.GetReading()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting reading: %v\n", err)
		return
	}

	if jsonOutput {
		jsonStr, err := reading.JSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			return
		}
		fmt.Println(jsonStr)
	} else {
		// Print a compact one-line format for polling
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] %s\n", timestamp, reading.ShortString())
	}
}
