package irprotocol // import "tinygo.org/x/drivers/irremote/irprotocol"

import (
	"fmt"
	"testing"

	qt "github.com/frankban/quicktest"
)

type NECTestData struct {
	Code    uint32
	Address uint16
	Command uint8
}

// Helper function to run NEC raw code decoding tests
func decodeTests(t *testing.T, tests []NECTestData, expectedValid bool) {
	c := qt.New(t)

	for _, data := range tests {
		name := fmt.Sprintf("Decode:Code:%08x Addr:%04x Cmd:%02x",
			data.Code, data.Address, data.Command)
		c.Run(name, func(c *qt.C) {
			valid, addr, cmd := SplitRawNECData(data.Code)
			c.Assert(valid, qt.Equals, expectedValid)
			if valid {
				c.Assert(addr, qt.Equals, data.Address)
				c.Assert(cmd, qt.Equals, data.Command)
			}
		})
	}
}

// Helper function to run NEC raw code encoding tests
func encodeTests(t *testing.T, tests []NECTestData) {
	c := qt.New(t)

	for _, data := range tests {
		name := fmt.Sprintf("Encode:Code:%08x Addr:%04x Cmd:%02x",
			data.Code, data.Address, data.Command)
		c.Run(name, func(c *qt.C) {
			code := MakeRawNECData(data.Address, data.Command)
			c.Assert(code, qt.Equals, data.Code)
		})
	}
}

// Tests encoding/decoding NEC raw data code with NEC non-extended (8-bit) addresses
func TestRawNECDataNonExtendedAddr(t *testing.T) {

	tests := []NECTestData{
		NECTestData{Code: 0xFF00FF00, Address: 0x0000, Command: 0x00},
		NECTestData{Code: 0x00FFFF00, Address: 0x0000, Command: 0xFF},
		NECTestData{Code: 0xFF0000FF, Address: 0x00FF, Command: 0x00},
		NECTestData{Code: 0x00FF00FF, Address: 0x00FF, Command: 0xFF},
		NECTestData{Code: 0xFF00DF20, Address: 0x0020, Command: 0x00},
		NECTestData{Code: 0xFF0020DF, Address: 0x00DF, Command: 0x00},
		NECTestData{Code: 0xDF20FF00, Address: 0x0000, Command: 0x20},
		NECTestData{Code: 0x20DFFF00, Address: 0x0000, Command: 0xDF},
	}
	decodeTests(t, tests, true)
	encodeTests(t, tests)
}

// Tests encoding/decoding NEC raw data code with NEC extended (16-bit) addresses
func TestRawNECDataExtendedAddr(t *testing.T) {
	tests := []NECTestData{
		NECTestData{Code: 0xFF000100, Address: 0x0100, Command: 0x00},
		NECTestData{Code: 0xFF00FE00, Address: 0xFE00, Command: 0x00},
		NECTestData{Code: 0xFF00F00D, Address: 0xF00D, Command: 0x00},
	}
	decodeTests(t, tests, true)
	encodeTests(t, tests)
}

// Tests decoding NEC raw data code with an invalid command verification
func TestSplitRawNECDataInvalidAddress(t *testing.T) {
	decodeTests(t,
		[]NECTestData{
			// Test single incorrect bit in each position of inverse command
			NECTestData{Code: 0x01FFFF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0x02FFFF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0x04FFFF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0x08FFFF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0x10FFFF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0x20FFFF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0x40FFFF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0x80FFFF00, Address: 0x0000, Command: 0xFF},
			// Test single incorrect bit in each position of command
			NECTestData{Code: 0xFF01FF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0xFF02FF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0xFF04FF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0xFF08FF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0xFF10FF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0xFF20FF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0xFF40FF00, Address: 0x0000, Command: 0xFF},
			NECTestData{Code: 0xFF80FF00, Address: 0x0000, Command: 0xFF},
		},
		false)
}
