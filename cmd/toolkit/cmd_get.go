package main

import (
	"fmt"
	"os"

	"github.com/skgsergio/tc66-toolkit/lib/tc66c"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a single reading from the device",
	Run: func(cmd *cobra.Command, args []string) {
		device := connectDevice(portFlag)
		defer device.Close()
		executeGet(device)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}

// executeGet gets a single reading from the device
func executeGet(tc66c *tc66c.TC66C) {
	reading, err := tc66c.GetReading()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting reading: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(reading.String())
}
