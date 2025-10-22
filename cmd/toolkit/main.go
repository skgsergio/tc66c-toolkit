package main

import (
	"fmt"
	"os"

	"github.com/skgsergio/tc66-toolkit/lib/tc66c"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	portFlag string
)

var rootCmd = &cobra.Command{
	Use:   "tc66c-toolkit",
	Short: "TC66C Toolkit - USB power meter interface",
	Long: `TC66C Toolkit provides a command-line interface for interacting with
TC66C USB power meters. You can read measurements, poll data continuously,
retrieve recordings, and update firmware.`,
}

func init() {
	// Disable the default help command (use --help flag instead)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// Global flags (available to all commands)
	rootCmd.PersistentFlags().StringVarP(&portFlag, "port", "p", "/dev/ttyACM0", "Serial port device path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// connectDevice connects to the TC66C device on the specified port
func connectDevice(port string) *tc66c.TC66C {
	fmt.Fprintf(os.Stderr, "Connecting to TC66C on %s...\n", port)
	device, err := tc66c.NewTC66C(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Connected successfully! Device mode: %s\n", device.Mode)
	return device
}
