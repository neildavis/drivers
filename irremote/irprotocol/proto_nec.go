package irprotocol // import "tinygo.org/x/drivers/irprotocol"

import "time"

// NEC protocol references
// https://www.sbprojects.net/knowledge/ir/nec.php
// https://techdocs.altium.com/display/FPGA/NEC+Infrared+Transmission+Protocol

const (
	// NEC Consumer IR is modulated at 38 kHz
	NEC_modulation_frequency = 38_000

	NEC_unit          = time.Nanosecond * 562_500 // 562.5 us
	NEC_lead_mark     = NEC_unit * 16             // 9 ms
	NEC_lead_space    = NEC_unit * 8              // 4.5 ms
	NEC_repeat_space  = NEC_unit * 4              // 2.25 ms
	NEC_bit_mark      = NEC_unit                  // 562.5 us
	NEC_bit_0_space   = NEC_unit                  // 562.5 us
	NEC_bit_1_space   = NEC_unit * 3              // 1.687 ms
	NEC_trail_mark    = NEC_unit                  // 562 us
	NEC_repeat_period = NEC_unit * 192            // 108 ms
)

// Helper func to break a raw NEC code into constituent parts performing validation
func SplitRawNECData(data uint32) (valid bool, address uint16, command byte) {
	valid = true
	addrLow := byte(data & 0xff)
	addrHigh := byte((data & 0xff00) >> 8)
	command = byte((data & 0xff0000) >> 16)
	invCmd := byte((data & 0xff000000) >> 24)
	address = MakeNECAddress(addrLow, addrHigh)
	// perform cmd inverse validation check
	if command != ^invCmd {
		// Validation failure. cmd and inverse cmd do not match
		valid = false
	}
	return
}

// Helper func to assemble a raw NEC code from constituent bytes
func MakeRawNECData(address uint16, command byte) uint32 {
	addrLow, addrHigh := SplitNECAddress(address)
	return (uint32(^command) << 24) | (uint32(command) << 16) | (uint32(addrHigh) << 8) | uint32(addrLow)
}

// Helper func to split an NEC address into low & high bytes
func SplitNECAddress(address uint16) (addrLow, addrHigh byte) {
	addrLow = byte(address & 0xff)
	addrHigh = byte((address & 0xff00) >> 8)
	if addrHigh == 0 {
		// NEC addresses in 8-bit range use inverse validation as addrHigh
		addrHigh = ^addrLow
	}
	return addrLow, addrHigh
}

// Helper func to assemble an NEC address from low & high bytes
func MakeNECAddress(addrLow, addrHigh byte) uint16 {
	if addrHigh == ^addrLow {
		// addrHigh is inverse of addrLow. This is not a valid 16-bit address in extended NEC coding
		// since it is indistinguishable from 8-bit address with inverse validation. Use the 8-bit address
		return uint16(addrLow)
	}
	return (uint16(addrHigh) << 8) | uint16(addrLow)
}
