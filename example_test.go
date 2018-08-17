package mid_test

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gomidi/midi/mid"
	"github.com/gomidi/midi/smf"
)

func Example() {
	// This example illustrates how the same handler can be used for live and SMF MIDI messages
	// It also illustrates how live and SMF midi can be written

	// make a SMF
	mkSMF := func() io.Reader {
		var bf bytes.Buffer
		wr := mid.NewSMFWriter(&bf, 1)
		wr.Tempo(160)
		wr.SetChannel(2)
		wr.NoteOn(65, 90)
		wr.SetDelta(4000)
		wr.NoteOff(65)
		wr.EndOfTrack()
		return bytes.NewReader(bf.Bytes())
	}

	hd := mid.NewReader(mid.NoLogger())

	// needed for the SMF timing
	var ticks smf.MetricTicks
	var bpm uint32 = 120 // default according to SMF spec

	// needed for the live timing
	var start = time.Now()

	// a helper to round the duration to seconds
	var roundSec = func(d time.Duration) time.Duration {
		return time.Second * time.Duration((d.Nanoseconds() / 1000000000))
	}

	// a helper to calculate the duration for both live and SMF messages
	var calcDuration = func(p *mid.SMFPosition) (dur time.Duration) {
		if p == nil {
			// we are in a live setting
			dur = roundSec(time.Now().Sub(start))
			return
		}

		// SMF data, calculate the time from the timeformat of the SMF file
		// we ignore the possibility that tempo information may come in a track following the one of
		// the current message as the spec does not recommend this
		return roundSec(ticks.Duration(bpm, uint32(p.AbsTime)))
	}

	hd.SMFHeader = func(head smf.Header) {
		// here we ignore that the timeformat could also be SMPTE
		ticks = head.TimeFormat.(smf.MetricTicks)
	}

	// we will override the tempo by the one given in the SMF
	hd.Message.Meta.Tempo = func(p mid.SMFPosition, valBPM uint32) {
		bpm = valBPM
	}

	// set the functions for the messages you are interested in
	hd.Message.Channel.NoteOn = func(p *mid.SMFPosition, channel, key, vel uint8) {
		fmt.Printf("[%vs] NoteOn at channel %v: key %v velocity: %v\n",
			calcDuration(p).Seconds(),
			channel, key, vel)
	}

	hd.Message.Channel.NoteOff = func(p *mid.SMFPosition, channel, key, vel uint8) {
		fmt.Printf("[%vs] NoteOff at channel %v: key %v velocity: %v\n",
			calcDuration(p).Seconds(),
			channel, key, vel)
	}

	// handle the smf
	fmt.Println("-- SMF data --")
	hd.ReadSMF(mkSMF())

	// handle the live data
	fmt.Println("-- live data --")
	lrd, lwr := io.Pipe()

	// WARNING this example does not deal with races and synchronization, it is just for illustration
	go func() {
		hd.ReadLive(lrd)
	}()

	mwr := mid.NewLiveWriter(lwr)

	// reset the time
	start = time.Now()

	mwr.SetChannel(11)

	// now write some live data
	mwr.NoteOn(120, 50)
	time.Sleep(time.Second * 2)
	mwr.NoteOff(120)
	time.Sleep(time.Second)

	// Output: -- SMF data --
	// [0s] NoteOn at channel 2: key 65 velocity: 90
	// [1s] NoteOff at channel 2: key 65 velocity: 0
	// -- live data --
	// [0s] NoteOn at channel 11: key 120 velocity: 50
	// [2s] NoteOff at channel 11: key 120 velocity: 0
}
