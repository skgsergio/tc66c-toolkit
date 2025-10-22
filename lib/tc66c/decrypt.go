package tc66c

import (
	"crypto/aes"
	"fmt"
)

// DecryptPacket decrypts the 192-byte encrypted packet using AES-ECB
func DecryptPacket(encrypted []byte) ([]byte, error) {
	if len(encrypted) != PacketSize {
		return nil, fmt.Errorf("invalid packet size: expected %d, got %d", PacketSize, len(encrypted))
	}

	// Create AES cipher with the static key
	block, err := aes.NewCipher(AESKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	decrypted := make([]byte, PacketSize)

	// ECB mode: decrypt each 16-byte block independently
	blockSize := block.BlockSize() // 16 bytes for AES
	for i := 0; i < len(encrypted); i += blockSize {
		block.Decrypt(decrypted[i:i+blockSize], encrypted[i:i+blockSize])
	}

	// Reorder blocks if necessary
	reordered, err := ReorderBlocks(decrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to reorder blocks: %w", err)
	}

	return reordered, nil
}

// ReorderBlocks detects block order and reorders them to pac1, pac2, pac3
func ReorderBlocks(data []byte) ([]byte, error) {
	if len(data) != PacketSize {
		return nil, fmt.Errorf("invalid data size: expected %d, got %d", PacketSize, len(data))
	}

	// Detect block positions
	blocks := make(map[string][]byte)
	blockPositions := make(map[string]int)

	for i := 0; i < NumBlocks; i++ {
		offset := i * BlockSize
		blockData := data[offset : offset+BlockSize]
		prefix := string(blockData[0:4])

		blocks[prefix] = blockData
		blockPositions[prefix] = i
	}

	// Verify all expected blocks are present
	if _, ok := blocks[Block1Prefix]; !ok {
		return nil, fmt.Errorf("missing %s block", Block1Prefix)
	}
	if _, ok := blocks[Block2Prefix]; !ok {
		return nil, fmt.Errorf("missing %s block", Block2Prefix)
	}
	if _, ok := blocks[Block3Prefix]; !ok {
		return nil, fmt.Errorf("missing %s block", Block3Prefix)
	}

	// Check if already in correct order
	if blockPositions[Block1Prefix] == 0 && blockPositions[Block2Prefix] == 1 && blockPositions[Block3Prefix] == 2 {
		return data, nil
	}

	// Reorder blocks
	reordered := make([]byte, PacketSize)
	copy(reordered[0:BlockSize], blocks[Block1Prefix])
	copy(reordered[BlockSize:2*BlockSize], blocks[Block2Prefix])
	copy(reordered[2*BlockSize:3*BlockSize], blocks[Block3Prefix])

	return reordered, nil
}

// CalculateCRC16Modbus calculates CRC-16/MODBUS checksum
func CalculateCRC16Modbus(data []byte) uint16 {
	var crc uint16 = 0xFFFF

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc = crc >> 1
			}
		}
	}

	return crc
}

// VerifyChecksum verifies the CRC-16/MODBUS checksum of decrypted data
// The checksum is stored as a zero-extended 32-bit field
func VerifyChecksum(data []byte, expectedCRC uint16) bool {
	calculatedCRC := CalculateCRC16Modbus(data)
	return calculatedCRC == expectedCRC
}
