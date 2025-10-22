package main

import (
	"fmt"
	"os"

	"github.com/skgsergio/tc66-toolkit/lib/tc66c"
	"github.com/spf13/cobra"
)

var getJSONFlag bool

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a single reading from the device",
	Run: func(cmd *cobra.Command, args []string) {
		device := connectDevice(portFlag)
		defer device.Close()
		executeGet(device, getJSONFlag)
	},
}

func init() {
	getCmd.Flags().BoolVarP(&getJSONFlag, "json", "j", false, "Output in JSON format")
	rootCmd.AddCommand(getCmd)
}

// executeGet gets a single reading from the device
func executeGet(tc66c *tc66c.TC66C, jsonOutput bool) {
	reading, err := tc66c.GetReading()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting reading: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		jsonStr, err := reading.JSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(jsonStr)
	} else {
		fmt.Fprintln(os.Stderr)
		fmt.Println(reading.String())
	}
}
