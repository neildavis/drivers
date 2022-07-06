package irremote // import "tinygo.org/x/drivers/irremote"

import (
	"machine"
	"time"
)

// PWM is used for the pulse distance modulation carrier of the IR signal
type PWM interface {
	Configure(config machine.PWMConfig) error
	Channel(pin machine.Pin) (channel uint8, err error)
	Top() uint32
	Set(channel uint8, value uint32)
}

// SenderDevice is the device for sending IR commands
type SenderDevice struct {
	pin   machine.Pin // IR LED pin
	pwm   PWM         // Modulation PWM
	pwmDC int         // Modulation Duty Cycle %
	chRpt chan int    // Channel used to signal end of auto-repeats
}

// SenderConfig is used to configure the SenderDevice
type SenderConfig struct {
	// Pin is the GPIO pin connected to the IR LED
	Pin machine.Pin
	// PWM is used for the IR modulation carrier signal on Pin
	PWM PWM
	// ModulationDutyCycle is the duty cycle (%) used for the PWM modulation carrier signal
	// A value of zero results in a duty cycle of 33%
	ModulationDutyCycle int
}

// NewSender returns a new IR sender device
func NewSender(config SenderConfig) SenderDevice {
	if config.ModulationDutyCycle < 1 ||
		config.ModulationDutyCycle > 100 {
		// Default duty cycle for modulation is 33%
		config.ModulationDutyCycle = 33
	}
	sender := SenderDevice{
		pin:   config.Pin,
		pwm:   config.PWM,
		pwmDC: config.ModulationDutyCycle}
	return sender
}

// Configure configures the output pin for the IR sender device
func (ir *SenderDevice) Configure() {
	ir.pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ir.pwm.Configure(machine.PWMConfig{Period: 1e9 / uint64(irp.NEC_modulation_frequency)})
}

// SendNEC sends a command using the NEC protocol.
// If autoRepeat is true, sender will continue to send repeat codes until cancelled via StopNECRepeats()
func (ir *SenderDevice) SendNEC(address uint16, command byte, autoRepeat bool) {
	// Package up raw data in NEC protocol format
	addrLow, addrHigh := SplitNECAddress(address)
	dataTxDuration := ir.SendNECRawBytes(addrLow, addrHigh, command, ^command)

	// Process repeats
	if autoRepeat {
		// Autorepeats are handled by a goroutine
		// We use a channel to allow the caller to signal us to stop by closing the channel.
		// No data is actually needed to be sent or received on the channel.
		ir.chRpt = make(chan int)
		go func(irs *SenderDevice) {
			// Wait for initial repeat period
			time.Sleep(nec_repeat_period - dataTxDuration)
			for irs.chRpt != nil {
				select {
				case <-irs.chRpt: // Channel has been closed. Cleanup & exit
					irs.chRpt = nil
				default: // Channel still open. Send the next repeat code
					repeatTxDuration := irs.SendNECRepeat()
					// Wait for next repeat period
					time.Sleep(nec_repeat_period - repeatTxDuration)
				}
			}
		}(ir)
	}
}

// SendNECRepeat can be used to manually send a repeat code using the NEC protocol.
// The Receiver will interpret this as a repeat of the last command sent.
// Caller is responsible for protocol timing. Consider using SendNEC() with autorepeat instead
// Returns the time taken to transmit
func (ir *SenderDevice) SendNECRepeat() time.Duration {

	ir.mark(nec_lead_mark)
	ir.space(nec_repeat_space)
	ir.mark(nec_trail_mark)

	return nec_lead_mark + nec_repeat_space + nec_trail_mark
}

// StopNECRepeats cancels any auto-repeat codes being generated after passing autoRepeat=true to SendNEC()
func (ir *SenderDevice) StopNECRepeats() {
	ir.waitForAutoRepeatCancel()
}

// SendNECRawCode is a low-level API that sends raw 32-bit data using the NEC protocol,
// Intended for advanced use cases (e.g. repeaters/relays/replays etc.)
// It is the caller's responsibility to ensure the 32-bit data packet is correctly assembled.
// LSB -> MSB: { address (Low), address (High), cmd, ^cmd }
// Returns the time taken to transmit (or zero if data is incorrectly assembled)
func (ir *SenderDevice) SendNECRawCode(data uint32) time.Duration {

	// Get constituent bytes of raw data to send
	valid, address, cmd := SplitRawNECData(data)
	if !valid {
		return 0
	}
	addrLow, addrHigh := SplitNECAddress(address)
	return ir.SendNECRawBytes(addrLow, addrHigh, cmd, ^cmd)
}

// SendNECRawBytes is a (very) low-level API that sends raw 32-bit data using the NEC protocol
// Intended for advanced use cases
// Returns the time taken to transmit
func (ir *SenderDevice) SendNECRawBytes(addrLow, addrHigh, cmd, invCmd byte) time.Duration {
	// If we are currently auto-repeating a previous code, cancel that
	ir.waitForAutoRepeatCancel()

	// NEC protocol requires us to send the bytes in this order
	bytesToSend := []byte{addrLow, addrHigh, cmd, invCmd}
	txDuration := nec_lead_mark + nec_lead_space + +32*nec_bit_mark + nec_trail_mark

	// Send lead marker & space
	ir.mark(nec_lead_mark)
	ir.space(nec_lead_space)

	// Send data
	for _, b := range bytesToSend {
		// We send bits ordered LSB -> MSB for each byte
		for i := 0; i < 8; i++ {
			mask := byte(1) << i
			ir.mark(nec_bit_mark)
			if b&mask == 0 {
				ir.space(nec_bit_0_space)
				txDuration += nec_bit_0_space
			} else {
				ir.space(nec_bit_1_space)
				txDuration += nec_bit_1_space
			}
		}
	}

	// Send tail marker to indicate end of data
	ir.mark(nec_trail_mark)

	return txDuration
}

func (ir *SenderDevice) waitForAutoRepeatCancel() {
	for ir.chRpt != nil {
		select {
		case <-ir.chRpt: // Channel has already been closed.
			// Wait for auto-repeat goroutine to cleanup & exit
			time.Sleep(nec_unit * 40)
		default: // Channel still open. Close it
			close(ir.chRpt)
		}
	}
}

func (ir *SenderDevice) mark(duration time.Duration) {
	// We have to pulse the carrier (using PWM) for duration
	pwmChan, _ := ir.pwm.Channel(ir.pin)
	ir.pwm.Set(pwmChan, ir.pwm.Top()*uint32(ir.pwmDC)/100) // duty cycle
	time.Sleep(duration)
	ir.pwm.Set(pwmChan, 0)
}

func (ir *SenderDevice) space(duration time.Duration) {
	// Since mark() always lowers the LED pin afterwards, there's nothing to do but wait
	time.Sleep(duration)
}
