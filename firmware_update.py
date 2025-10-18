#!/usr/bin/env python3
"""
TC66/TC66C firmware updater script
"""

import argparse
import math
import pathlib
import time

import serial


def send_data(
    ser: serial.Serial,
    data: str | bytes,
    response_length: int = 0,
) -> str | None:
    ser.write(data.encode() if isinstance(data, str) else data)

    return ser.read(response_length) if response_length > 0 else None


def firmware_update(port: str, firmware_file: str) -> bool:
    baudrate = 115200
    chunk_size = 64

    firmware = pathlib.Path(firmware_file)

    if not firmware.exists():
        print(f"[!] Firmware file '{firmware.name}' not found!")
        return False

    print(f"[*] Connecting to {port}...")

    ser = serial.Serial(
        port=port,
        baudrate=baudrate,
        timeout=2,
        write_timeout=2,
    )

    print("\n[*] Check if running in bootloader mode...")

    if (response := send_data(ser, "query", response_length=4)) != b"boot":
        print(f"[!] Device replied with '{response.decode()}', expected 'boot'!")
        print("[!] Press K1 while plugging in to enter bootloader mode.")
        ser.close()
        return False

    print("[+] OK!")

    print("\n[*] Entering in firmware update mode...")

    if (response := send_data(ser, "update", response_length=5)) != b"uprdy":
        print(f"[!] Device replied with '{response.decode()}', expected 'uprdy'!")
        print("T[!] Try again after unplugging and plugging the device back in.")
        ser.close()
        return False

    print("[+] OK!")

    print(f"\n[*] Sending firmware file '{firmware}'...")

    file_size = firmware.stat().st_size
    chunk_count = math.ceil(file_size / chunk_size)

    print(f"[+] File size: {file_size} bytes ({chunk_count} {chunk_size}-bytes chunks)")

    bytes_sent = 0
    chunks_sent = 0
    with firmware.open("rb") as f:
        while chunk := f.read(chunk_size):
            if not chunk:
                break

            chunks_sent += 1
            bytes_sent += len(chunk)

            if (response := send_data(ser, chunk, response_length=2)) != b"OK":
                print()
                print(f"[!] Device replied with '{response.decode()}', expected 'OK'!")
                print("[!] Your device may not boot normally in this state, try again.")
                ser.close()
                return False

            progress = (bytes_sent / file_size) * 100
            print(f"[>] Progress: {bytes_sent}/{file_size} bytes ({progress:.0f}%) - Chunk {chunks_sent}/{chunk_count} OK", end="\r")

    print("\n\n[*] Firmware update completed successfully!")

    ser.close()
    print("\n[+] Serial connection closed.")

    return True


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="TC66/TC66C firmware updater",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Example usage:
  %(prog)s -p /dev/ttyACM0 -f TC66.bin
  %(prog)s --port /dev/ttyUSB0 --file firmware.bin
        """,
    )

    parser.add_argument(
        "-p", "--port",
        type=str,
        required=True,
        help="Serial port to connect to (e.g., /dev/ttyACM0, /dev/ttyUSB0)",
    )

    parser.add_argument(
        "-f", "--file",
        type=str,
        required=True,
        help="Firmware binary file to send",
    )

    args = parser.parse_args()

    firmware_update(args.port, args.file)