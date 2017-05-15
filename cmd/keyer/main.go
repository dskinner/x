// Command keyer experiments with keyboard chording at system level.
package main

/*
#include <linux/input.h>
#include <linux/uinput.h>
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	flagDisable = flag.String("disable", "", "xinput device id to disable during execution.")
	flagLog     = flag.Bool("log", false, "prints logging to stdout")
)

var (
	btou16 = binary.LittleEndian.Uint16
	btou32 = binary.LittleEndian.Uint32
	btou64 = binary.LittleEndian.Uint64
	putu16 = binary.LittleEndian.PutUint16
	putu32 = binary.LittleEndian.PutUint32
	putu64 = binary.LittleEndian.PutUint64
)

func ioctl(fd uintptr, request uintptr, argp uintptr) error {
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, request, argp); errno != 0 {
		return os.NewSyscallError("ioctl", errno)
	}
	return nil
}

type Device struct {
	Name    string
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16

	f *os.File
}

func (dev *Device) Open(keybits ...uintptr) (err error) {
	dev.f, err = os.OpenFile("/dev/uinput", unix.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return err
	}

	if err = ioctl(dev.f.Fd(), C.UI_SET_EVBIT, C.EV_SYN); err != nil {
		return err
	}

	if err = ioctl(dev.f.Fd(), C.UI_SET_EVBIT, C.EV_KEY); err != nil {
		return err
	}

	for _, code := range keybits {
		if err = ioctl(dev.f.Fd(), C.UI_SET_KEYBIT, code); err != nil {
			return err
		}
	}

	bin, err := dev.MarshalBinary()
	if err != nil {
		return err
	}

	if _, err := dev.f.Write(bin); err != nil {
		return err
	}

	return ioctl(dev.f.Fd(), C.UI_DEV_CREATE, 0)
}

func (dev *Device) Close() {
	if dev.f != nil {
		_ = ioctl(dev.f.Fd(), C.UI_DEV_DESTROY, 0)
		_ = dev.f.Close()
	}
}

func (dev *Device) Write(ev Event) error {
	bin, err := ev.MarshalBinary()
	if err != nil {
		return err
	}
	if _, err := dev.f.Write(bin); err != nil {
		return err
	}
	return nil
}

func (dev *Device) MarshalBinary() ([]byte, error) {
	if C.sizeof_struct_input_id != 8 {
		return nil, errors.New("unexpected struct size for input_id")
	}
	bin := make([]byte, C.sizeof_struct_uinput_user_dev)
	copy(bin[:C.UINPUT_MAX_NAME_SIZE], dev.Name)

	buf := bin[C.UINPUT_MAX_NAME_SIZE:]
	putu16(buf, dev.Bustype)
	putu16(buf[2:], dev.Vendor)
	putu16(buf[4:], dev.Product)
	putu16(buf[6:], dev.Version)

	return bin, nil
}

type Event struct {
	Time struct {
		Unix  int64
		Micro int64
	}
	Type  uint16
	Code  uint16
	Value int32
}

func (ev Event) MarshalBinary() ([]byte, error) {
	if C.sizeof_struct_input_event != 24 {
		return nil, errors.New("unexpected struct size for input_event")
	}
	bin := make([]byte, 24)
	putu64(bin, uint64(ev.Time.Unix))
	putu64(bin[8:], uint64(ev.Time.Micro))
	putu16(bin[16:], ev.Type)
	putu16(bin[18:], ev.Code)
	putu32(bin[20:], uint32(ev.Value))
	return bin, nil
}

func (ev *Event) UnmarshalBinary(bin []byte) error {
	if len(bin) != 24 {
		return errors.New("unexpected bin length to unmarshal Event")
	}
	ev.Time.Unix = int64(btou64(bin))
	ev.Time.Micro = int64(btou64(bin[8:]))
	ev.Type = btou16(bin[16:])
	ev.Code = btou16(bin[18:])
	ev.Value = int32(btou32(bin[20:]))
	return nil
}

var runemap = map[rune]uintptr{
	'a': C.KEY_A,
	'b': C.KEY_B,
	'c': C.KEY_C,
	'd': C.KEY_D,
	'e': C.KEY_E,
	'f': C.KEY_F,
	'g': C.KEY_G,
	'h': C.KEY_H,
	'i': C.KEY_I,
	'j': C.KEY_J,
	'k': C.KEY_K,
	'l': C.KEY_L,
	'm': C.KEY_M,
	'n': C.KEY_N,
	'o': C.KEY_O,
	'p': C.KEY_P,
	'q': C.KEY_Q,
	'r': C.KEY_R,
	's': C.KEY_S,
	't': C.KEY_T,
	'u': C.KEY_U,
	'v': C.KEY_V,
	'w': C.KEY_W,
	'x': C.KEY_X,
	'y': C.KEY_Y,
	'z': C.KEY_Z,
	' ': C.KEY_SPACE,
	';': C.KEY_SEMICOLON,
	'.': C.KEY_DOT,
}

var keymap = map[uint8]rune{
	// 1000
	0x88: 'a',
	0x48: 's',
	0x28: 'd',
	0x18: 'f',
	// 0100
	0x84: 'q',
	0x44: 'w',
	0x24: 'e',
	0x14: 'r',
	// 0010
	0x82: 'u',
	0x42: 'i',
	0x22: 'o',
	0x12: 'p',
	// 0001
	0x81: 't',
	0x41: 'y',
	0x21: 'g',
	0x11: 'h',
	// 1100
	0x8C: 'v',
	0x4C: 'b',
	0x2C: 'n',
	0x1C: 'm',
	// 0110
	0x86: 'j',
	0x46: 'k',
	0x26: 'l',
	0x16: ';',
	// 1010
	0x8A: '.',
	0x4A: 'z',
	0x2A: 'x',
	0x1A: 'c',
}

func main() {
	flag.Parse()

	if *flagDisable != "" {
		if err := exec.Command("xinput", "--disable", *flagDisable).Run(); err != nil {
			log.Fatal(err)
		}
		defer exec.Command("xinput", "--enable", *flagDisable).Run()
	} else {
		var state unix.Termios
		if err := ioctl(uintptr(unix.Stdin), unix.TCGETS, uintptr(unsafe.Pointer(&state))); err != nil {
			log.Fatal(err)
		}
		newstate := state
		newstate.Lflag &^= unix.ECHO
		if err := ioctl(uintptr(unix.Stdin), unix.TCSETS, uintptr(unsafe.Pointer(&newstate))); err != nil {
			log.Fatal(err)
		}
		defer ioctl(uintptr(unix.Stdin), unix.TCSETS, uintptr(unsafe.Pointer(&state)))
	}

	dev := &Device{
		Name:    "keyer",
		Bustype: C.BUS_USB,
		Vendor:  0x1234,
		Product: 0xfedc,
		Version: 1,
	}

	keybits := []uintptr{C.KEY_ESC}
	for _, v := range runemap {
		keybits = append(keybits, v)
	}

	if err := dev.Open(keybits...); err != nil {
		log.Fatal(err)
	}
	defer dev.Close()

	f, err := os.OpenFile("/dev/input/event0", unix.O_RDONLY, 0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Press ESC to exit")
	var state uint8
	for {
		bin := make([]byte, 24)
		_, err := f.Read(bin)
		if err != nil {
			log.Println(err)
			continue
		}
		var ev Event
		ev.UnmarshalBinary(bin)

		if ev.Type != C.EV_KEY {
			continue
		}

		var mask uint8
		switch ev.Code {
		case C.KEY_ESC:
			return
		case C.KEY_SPACE:
			dev.Write(ev)
			dev.Write(Event{Type: C.EV_SYN})
			continue
		case C.KEY_A:
			mask = 1 << 7
		case C.KEY_S:
			mask = 1 << 6
		case C.KEY_D:
			mask = 1 << 5
		case C.KEY_F:
			mask = 1 << 4
		case C.KEY_J:
			mask = 1 << 3
		case C.KEY_K:
			mask = 1 << 2
		case C.KEY_L:
			mask = 1 << 1
		case C.KEY_SEMICOLON:
			mask = 1
		}

		switch ev.Value {
		case 0:
			state &^= mask
			continue
		case 1, 2:
			if (mask >> 4) != 0 {
				state &^= 0xF0
			}
			state |= mask
		}

		if *flagLog {
			fmt.Printf("%08[1]b: 0x%[1]X\n", state)
		}

		if r, ok := keymap[state]; ok {
			if *flagDisable == "" {
				fmt.Print(string(r))
			} else {
				if code, ok := runemap[r]; ok {
					if err := dev.Write(Event{Type: C.EV_KEY, Code: uint16(code), Value: 1}); err != nil {
						log.Println(err)
					}
					if err := dev.Write(Event{Type: C.EV_KEY, Code: uint16(code), Value: 0}); err != nil {
						log.Println(err)
					}
					if err := dev.Write(Event{Type: C.EV_SYN}); err != nil {
						log.Println(err)
					}
				}
			}
		}
	}
}
