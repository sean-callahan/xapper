package xapper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
)

type DeviceType byte

// Implemented device types
const (
	PSR1212 DeviceType = 4
	XAP800  DeviceType = 5
	XAPTH2  DeviceType = 6
	XAP400  DeviceType = 7
)

// Inputs returns the number of input channels this device supports.
func (dt DeviceType) Inputs() int {
	switch dt {
	case PSR1212:
	case XAP800:
		return 12
	case XAP400:
		return 8
	case XAPTH2:
		return 0
	}
	return -1
}

// Outputs returns the number of output channels this device supports.
func (dt DeviceType) Outputs() int {
	switch dt {
	case PSR1212:
	case XAP800:
		return 12
	case XAP400:
		return 9
	case XAPTH2:
		return 0
	}
	return -1
}

// Group identifies a channel group
type Group string

// Channel groups
const (
	Input  Group = "I"
	Output Group = "O"
)

func (g Group) MarshalJSON() ([]byte, error) {
	var s string
	switch g {
	default:
		s = "unknown"
	case Input:
		s = "input"
	case Output:
		s = "output"
	}
	return json.Marshal(s)
}

type Device struct {
	Type     DeviceType           `json:"-"`
	ID       int                  `json:"-"`
	Channels map[Group][]*Channel `json:"channels"`

	UID     string `json:"-"`
	Version string `json:"-"`

	port   *serial.Port
	cmdBuf *bytes.Buffer
	mu     sync.Mutex
	logger *log.Logger
}

func NewDevice(id int, typ DeviceType, c *serial.Config, logger *log.Logger) (d *Device, err error) {
	d = new(Device)
	d = &Device{
		ID:       id,
		Type:     typ,
		Channels: make(map[Group][]*Channel),
		cmdBuf:   new(bytes.Buffer),
		logger:   logger,
	}

	d.port, err = serial.OpenPort(c)
	if err != nil {
		return nil, err
	}

	d.Version, err = d.Send("VER")
	if err != nil {
		return nil, err
	}

	d.UID, err = d.Send("UID")
	if err != nil {
		return nil, err
	}

	for i := 0; i < 12; i++ {
		d.Channels[Input] = append(d.Channels[Input], &Channel{Number: i + 1, Group: Input, d: d})
		go d.Channels[Input][i].Update()
		d.Channels[Output] = append(d.Channels[Output], &Channel{Number: i + 1, Group: Output, d: d})
		go d.Channels[Output][i].Update()
	}

	return d, nil
}

func (d *Device) Start(heartbeat time.Duration) {
	go func(d *Device, ival time.Duration) {
		for {
			time.Sleep(ival)
			for _, channels := range d.Channels {
				for _, ch := range channels {
					if !ch.Muted {
						ch.Heartbeat()
					}
				}
			}
		}
	}(d, heartbeat)
}

func (d *Device) Close() error {
	if d.port == nil {
		return nil
	}
	return d.port.Close()
}

func (d *Device) Send(cmd string, args ...string) (resp string, err error) {
	d.mu.Lock()
	defer func() {
		d.cmdBuf.Reset()
		d.mu.Unlock()
	}()

	d.cmdBuf.WriteString(fmt.Sprintf("#%d%d %s", d.Type, d.ID, cmd))

	for _, arg := range args {
		d.cmdBuf.WriteByte(' ')
		d.cmdBuf.WriteString(arg)
	}

	if d.logger != nil {
		d.logger.Println("Tx:", string(d.cmdBuf.String()))
	}

	d.cmdBuf.WriteString("\r\n")

	_, err = d.port.Write(d.cmdBuf.Bytes())
	if err != nil {
		return "", err
	}

	buf := make([]byte, 128)
	i := 0

	var n int

	start := -1
	end := -1

	for {
		n, err = d.port.Read(buf[i:])
		if err != nil {
			return "", err
		}

		if buf[i] == '#' {
			start = i
		}
		if (start >= 0 && buf[i] == '\n') || n == 0 || buf[i] == 0x00 {
			end = i
			break
		}
		i += n
	}

	if start >= 0 && end >= 0 {
		resp = strings.TrimSpace(string(buf[start+3 : end+1]))
	} else {
		return "", errors.New("no response")
	}

	erri := strings.Index(resp, "ERROR ")
	if erri >= 0 {
		return "", errors.New(resp[erri+6:])
	}

	if d.logger != nil {
		d.logger.Println("Rx:", resp)
	}

	return resp, nil
}

type Channel struct {
	Number int     `json:"-"`
	Group  Group   `json:"-"`
	Label  string  `json:"label"`
	Muted  bool    `json:"muted"`
	Gain   float32 `json:"gain"`
	Level  float32 `json:"level"`

	d *Device

	mu sync.RWMutex
}

func (ch *Channel) Update() error {
	_, err := ch.mute("")
	if err != nil {
		return err
	}

	_, err = ch.gain("")
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = ch.label("")
	if err != nil {
		return err
	}

	return nil
}

func (ch *Channel) Heartbeat() error {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	_, err := ch.level("")
	if err != nil {
		return err
	}
	return nil
}

func (ch *Channel) mute(value string) (bool, error) {
	resp, err := ch.d.Send("MUTE", strconv.Itoa(ch.Number), string(ch.Group), value)
	if err != nil {
		return false, err
	}

	parts := strings.Split(resp, " ")
	if len(parts) != 4 {
		return false, errors.New("invalid response")
	}

	v := parts[3]

	muted := false

	if v == "1" {
		muted = true
	}

	ch.Muted = muted

	return muted, nil
}

func (ch *Channel) Mute(mute bool) (bool, error) {
	v := "0"
	if mute {
		v = "1"
	}
	return ch.mute(v)
}

func (ch *Channel) label(value string) (string, error) {
	resp, err := ch.d.Send("LABEL", strconv.Itoa(ch.Number), string(ch.Group), value)
	if err != nil {
		return "", err
	}
	parts := strings.Split(resp, " ")
	if len(parts) != 5 {
		return "", errors.New("invalid response")
	}
	ch.Label = parts[3]
	return ch.Label, nil
}

func (ch *Channel) gain(value string) (float32, error) {
	var resp string
	var err error

	if value == "" {
		resp, err = ch.d.Send("GAIN", strconv.Itoa(ch.Number), string(ch.Group), "")
	} else {
		resp, err = ch.d.Send("GAIN", strconv.Itoa(ch.Number), string(ch.Group), value, "A")
	}
	if err != nil {
		return 0, err
	}

	parts := strings.Split(resp, " ")
	if len(parts) != 5 {
		return 0, errors.New("invalid response")
	}
	value = parts[3]

	gain, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, err
	}

	ch.Gain = float32(gain)

	return ch.Gain, nil
}

func (ch *Channel) SetGain(value float32) (float32, error) {
	return ch.gain(fmt.Sprintf("%f", value))
}

func (ch *Channel) level(value string) (float32, error) {
	resp, err := ch.d.Send("LVL", strconv.Itoa(ch.Number), string(ch.Group), "A", value)
	if err != nil {
		return 0, err
	}

	parts := strings.Split(resp, " ")
	if len(parts) != 5 {
		return 0, errors.New("invalid response")
	}
	value = parts[4]

	level, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, err
	}

	ch.Level = float32(level)

	return ch.Level, nil
}
