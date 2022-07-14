// Simple Midi Interface for Novation Launchpad.
package pad

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/scgolang/midi"
)

type Pad struct {
	*midi.Device
	hits chan Hit
}
type Hit struct {
	X   uint8
	Y   uint8
	Err error
}

func Open() (*Pad, error) {
	devices, err := midi.Devices()
	if err != nil {
		return nil, errors.Wrap(err, "listing MIDI devices")
	}
	var device *midi.Device
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), "Pad") {
			device = d
			break
		}
	}
	if device == nil {
		return nil, errors.New("Pad not found")
	}
	pad := &Pad{Device: device}
	if err := pad.Open(); err != nil {
		return nil, err
	}
	return pad, nil
}

// Close closes the connection to the Pad.
func (pad *Pad) Close() error {
	if pad.hits != nil {
		close(pad.hits)
	}
	return errors.Wrap(pad.Device.Close(), "closing midi device")
}

// Hits returns a channel that emits when the Pad buttons are hit.
func (pad *Pad) Hits() (<-chan Hit, error) {
	if pad.hits != nil {
		return pad.hits, nil
	}
	packets, err := pad.Packets()
	if err != nil {
		return nil, errors.Wrap(err, "getting packets channel")
	}
	hits := make(chan Hit)

	packet := <-packets

	go convert(packet, hits)
	pad.hits = hits
	return hits, nil
}

func (pad *Pad) Reset() error {
	_, err := pad.Write([]byte{0xb0, 0, 0})
	return err
}

func convert(packets []midi.Packet, hits chan<- Hit) {
	for _, packet := range packets {
		if packet.Err != nil {
			hits <- Hit{Err: packet.Err}
			continue
		}
		if packet.Data[2] == 0 {
			continue
		}
		var x, y uint8

		if packet.Data[0] == 176 {
			x = packet.Data[1] - 104
			y = 8
		} else if packet.Data[0] == 144 {
			x = packet.Data[1] % 16
			y = (packet.Data[1] - x) / 16
		} else {
			continue
		}
		hits <- Hit{X: x, Y: y}
	}
}
