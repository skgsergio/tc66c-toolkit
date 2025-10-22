package main

import (
	"fmt"
	"os"

	"github.com/skgsergio/tc66-toolkit/lib/tc66c"
	"github.com/spf13/cobra"
)

var firmwareFileFlag string

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update device firmware (requires bootloader mode)",
	Long: `Update the device firmware from a binary file.

The device must be in bootloader mode before running this command.
To enter bootloader mode:
  1. Unplug the device
  2. Press and hold the K1 button
  3. While holding K1, plug in the device
  4. Release K1`,
	Run: func(cmd *cobra.Command, args []string) {
		if firmwareFileFlag == "" {
			fmt.Fprintf(os.Stderr, "Error: -file flag is required\n")
			cmd.Usage()
			os.Exit(1)
		}
		device := connectDevice(portFlag)
		defer device.Close()
		executeUpdate(device, firmwareFileFlag)
	},
}

func init() {
	updateCmd.Flags().StringVarP(&firmwareFileFlag, "file", "f", "", "Firmware file (required)")
	rootCmd.AddCommand(updateCmd)
}

// executeUpdate updates the device firmware
func executeUpdate(device *tc66c.TC66C, firmwareFile string) {
	// Check if device is in bootloader mode
	if device.Mode != tc66c.ModeBootloader {
		fmt.Fprintf(os.Stderr, "Error: Device must be in bootloader mode to update firmware\n")
		fmt.Fprintf(os.Stderr, "Current mode: %s\n", device.Mode)
		fmt.Fprintf(os.Stderr, "\nTo enter bootloader mode:\n")
		fmt.Fprintf(os.Stderr, "1. Unplug the device\n")
		fmt.Fprintf(os.Stderr, "2. Press and hold the K1 button\n")
		fmt.Fprintf(os.Stderr, "3. While holding K1, plug in the device\n")
		fmt.Fprintf(os.Stderr, "4. Release K1\n")
		os.Exit(1)
	}

	// Read firmware file
	fmt.Printf("Reading firmware file '%s'...\n", firmwareFile)
	firmwareData, err := os.ReadFile(firmwareFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading firmware file: %v\n", err)
		os.Exit(1)
	}

	fileSize := len(firmwareData)
	chunkCount := (fileSize + tc66c.FirmwareChunkSize - 1) / tc66c.FirmwareChunkSize
	fmt.Printf("Firmware file size: %d bytes (%d chunks of %d bytes)\n\n", fileSize, chunkCount, tc66c.FirmwareChunkSize)

	fmt.Println("WARNING: Do not disconnect the device during the update!")
	fmt.Println("Starting firmware update...")
	fmt.Println()

	// Update firmware with progress callback
	err = device.UpdateFirmware(firmwareData, func(progress tc66c.FirmwareUpdateProgress) {
		percentage := float64(progress.BytesSent) / float64(progress.TotalBytes) * 100
		fmt.Printf("\r[>] Progress: %d/%d bytes (%.0f%%) - Chunk %d/%d OK",
			progress.BytesSent, progress.TotalBytes, percentage,
			progress.ChunksSent, progress.TotalChunks)
	})

	fmt.Println() // New line after progress

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: Firmware update failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nWARNING: Your device may not boot normally in this state.\n")
		fmt.Fprintf(os.Stderr, "Try running the update again. If it still fails, you may need to\n")
		fmt.Fprintf(os.Stderr, "use recovery procedures specific to your device.\n")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Firmware update completed successfully!")
	fmt.Println()
	fmt.Println("You can now unplug and replug the device to boot into the new firmware.")
}
