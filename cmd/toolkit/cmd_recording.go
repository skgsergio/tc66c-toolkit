package main

import (
	"fmt"
	"os"

	"github.com/skgsergio/tc66-toolkit/lib/tc66c"
	"github.com/spf13/cobra"
)

var recordingCmd = &cobra.Command{
	Use:   "recording",
	Short: "Retrieve recordings from the device",
	Run: func(cmd *cobra.Command, args []string) {
		device := connectDevice(portFlag)
		defer device.Close()
		executeRecording(device)
	},
}

func init() {
	rootCmd.AddCommand(recordingCmd)
}

// executeRecording retrieves recordings from the device
func executeRecording(tc66c *tc66c.TC66C) {
	fmt.Println("Retrieving recordings...")

	recordings, err := tc66c.GetRecordings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting recordings: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Received %d recording entries\n\n", len(recordings))

	if len(recordings) == 0 {
		fmt.Println("No recordings available")
		return
	}

	// Print header
	fmt.Printf("%-6s | %-12s | %-12s\n", "Index", "Voltage (V)", "Current (A)")
	fmt.Println("-------+-------------+-------------")

	// Print each recording entry
	for i, entry := range recordings {
		fmt.Printf("%-6d | %10.4f V | %10.5f A\n", i, entry.Voltage, entry.Current)
	}

	fmt.Printf("\nTotal entries: %d\n", len(recordings))
}
