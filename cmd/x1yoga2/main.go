package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	sensors, pointers   []string
	scale               float64
	xprop, yprop, zprop Prop

	orient Orientation

	flagVerbose = flag.Bool("v", false, "verbose")
)

func init() {
	flag.Parse()

	log.SetFlags(0)
	if !*flagVerbose {
		log.SetOutput(ioutil.Discard)
	}

	sensors, pointers = xinputs()
	if len(sensors) == 0 {
		panic("failed to locate sensors via xinput")
	}
	if len(pointers) == 0 {
		panic("failed to locate pointers via xinput")
	}

	dev, err := device("incli_3d")
	if err != nil {
		panic(err)
	}
	scale = Prop(filepath.Join(dev, "in_incli_scale")).Val()
	xprop = Prop(filepath.Join(dev, "in_incli_x_raw"))
	yprop = Prop(filepath.Join(dev, "in_incli_y_raw"))
	zprop = Prop(filepath.Join(dev, "in_incli_z_raw"))
}

func xinputs() (sensors, pointers []string) {
	cmd := exec.Command("xinput", "list", "--name-only")
	bin, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	for _, name := range strings.Split(string(bin), "\n") {
		if name == "Virtual core keyboard" {
			return sensors, pointers
		}
		if strings.Contains(name, "sensor") {
			sensors = append(sensors, name)
		}
		if strings.Contains(name, "TrackPoint") {
			pointers = append(pointers, name)
		}
		if strings.Contains(name, "TouchPad") {
			// TODO don't manage if already disabled.
			pointers = append(pointers, name)
		}
	}
	// must return at "Virtual core keyboard"
	panic("failed to read xinput list correctly")
}

// device returns sys path for iio device by name or error if not exists.
func device(name string) (string, error) {
	const devices = "/sys/bus/iio/devices/"
	dirs, err := ioutil.ReadDir(devices)
	if err != nil {
		return "", err
	}
	for _, dir := range dirs {
		dpath := filepath.Join(devices, dir.Name())
		bin, err := ioutil.ReadFile(filepath.Join(dpath, "name"))
		if err != nil {
			log.Println(err)
		} else if dname := string(bytes.TrimSpace(bin)); dname == name {
			return dpath, nil
		}
	}
	return "", os.ErrNotExist
}

type Prop string

func (p Prop) Val() float64 {
	bin, err := ioutil.ReadFile(string(p))
	if err != nil {
		panic(err)
	}
	if len(bin) > 0 {
		// drop assumed \n
		bin = bin[:len(bin)-1]
	}
	v, err := strconv.ParseFloat(string(bin), 64)
	if err != nil {
		panic(err)
	}
	return v
}

type Orientation int

const (
	Normal Orientation = iota
	Left
	Inverted
	Right
)

func (n Orientation) String() string {
	switch n {
	case Normal:
		return "normal"
	case Left:
		return "left"
	case Inverted:
		return "inverted"
	case Right:
		return "right"
	default:
		panic("unknown orientation")
	}
}

func (n Orientation) Transform() []string {
	switch n {
	case Normal:
		return []string{"1", "0", "0", "0", "1", "0", "0", "0", "1"}
	case Left:
		return []string{"0", "-1", "1", "1", "0", "0", "0", "0", "1"}
	case Inverted:
		return []string{"-1", "0", "1", "0", "-1", "1", "0", "0", "1"}
	case Right:
		return []string{"0", "1", "0", "-1", "0", "1", "0", "0", "1"}
	default:
		panic("unknown orientation")
	}
}

func (n Orientation) Case() string {
	switch n {
	case Normal:
		return "--enable"
	case Left, Inverted, Right:
		return "--disable"
	default:
		panic("unknown orientation")
	}
}

func (n Orientation) Do() {
	if err := exec.Command("xrandr", "-o", n.String()).Run(); err != nil {
		log.Println(err)
	}

	for _, name := range sensors {
		cmd := exec.Command("xinput", "set-prop", name, "Coordinate Transformation Matrix")
		cmd.Args = append(cmd.Args, n.Transform()...)
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
	}

	for _, name := range pointers {
		if err := exec.Command("xinput", n.Case(), name).Run(); err != nil {
			log.Println(err)
		}
	}
}

func (n *Orientation) Set(x, y, z float64) bool {
	if o := Orient3D(x, y, z); *n != o {
		*n = o
		return true
	}
	return false
}

func Orient3D(x, y, z float64) Orientation {
	x0 := -0.5 < x && x < 0.5
	x1 := x < -2.5 || 2.5 < x

	switch {
	case y < -0.5 && x0:
		return Right
	case y < -0.5 && x1:
		return Left
	case 0.5 < y && x0:
		return Left
	case 0.5 < y && x1:
		return Right
	case x < 0:
		return Inverted
	default:
		return Normal
	}
}

func main() {
	for range time.Tick(time.Second) {
		x := scale * xprop.Val()
		y := scale * yprop.Val()
		z := scale * zprop.Val()

		log.Printf("\nIncli3D %s\nx: %+.2f\ny: %+.2f\nz: %+.2f\n", orient, x, y, z)

		if orient.Set(x, y, z) {
			orient.Do()
			log.Println("ORIENTED")
		}
	}
}
