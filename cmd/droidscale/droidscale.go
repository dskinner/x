// Command droidscale resizes android resources.
//
// Set srcdpi to ldpi, mdpi, hdpi, xhdpi, xxhdpi and xxxhdpi; srcdpi scaling is
// skipped by default. Specifying a srcdpi below xxxhdpi will scale images up
// if -scaleup flag is set. Contents of flag path will be walked and new files
// written to out directory will be flattened. This means you can organize the
// contents of flag path with subdirectories to target different interpolations
// for use on a particular set of images. For example:
//
//   droidscale -path="xhdpi/L2" -interp="Lanzcos2"
//   droidscale -path="xhdpi/L3" -interp="Lanzcos3"
//
// Inspecting the contents of ./out/drawable-mdpi/ and friends will show a flat
// list of the resized files.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

var (
	flagPath    = flag.String("path", "", "path of xhdpi drawables to process, can point to a single file.")
	flagOut     = flag.String("out", "./out", "directory to place resized images.")
	flagSrcDpi  = flag.String("srcdpi", "xxxhdpi", "source dpi of images to be scaled.")
	flagInterp  = flag.String("interp", "Lanzcos3", "interpolation to use. NearestNeighbor, Bilinear, Bicubic, MitchellNetravali, Lanzcos2, Lanzcos3.")
	flagLDPI    = flag.Bool("ldpi", false, "force output of ldpi.")
	flagScaleUp = flag.Bool("scaleup", false, "will generate assets at higher dpi than srcdpi if true.")

	srcdpi dpi
	interp resize.InterpolationFunction
)

// dpi represents a density by its mdpi scale factor.
type dpi float64

// scale takes an image dimension and makes suitable for pkg resize.
func (d dpi) scale(n int) uint {
	s := float64(d) / float64(srcdpi)
	return uint(float64(n) * s)
}

const (
	ldpi    dpi = 0.75
	mdpi        = 1
	hdpi        = 1.5
	xhdpi       = 2
	xxhdpi      = 3
	xxxhdpi     = 4
)

var dpiString = map[dpi]string{
	ldpi:    "drawable-ldpi",
	mdpi:    "drawable-mdpi",
	hdpi:    "drawable-hdpi",
	xhdpi:   "drawable-xhdpi",
	xxhdpi:  "drawable-xxhdpi",
	xxxhdpi: "drawable-xxxhdpi",
}

// visit is a filepath.WalkFn that handles finding and resizing png resources.
func visit(path string, f os.FileInfo, err error) (e error) {
	if filepath.Ext(path) != ".png" {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Failed to open file %s\n", path)
		return
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		fmt.Printf("Failed to decode png %s\n", path)
		return
	}

	fn := func(factor dpi) {
		if factor == srcdpi {
			return
		}
		if !*flagScaleUp && factor > srcdpi {
			return
		}
		x := factor.scale(img.Bounds().Size().X)
		m := resize.Resize(x, 0, img, interp)
		p := filepath.Join(*flagOut, dpiString[factor], f.Name())
		d := filepath.Dir(p)
		os.MkdirAll(d, 0766)
		out, err := os.Create(p)
		if err != nil {
			fmt.Printf("Failed to create new file %s\n", p)
			return
		}
		defer out.Close()
		if err := png.Encode(out, m); err != nil {
			fmt.Printf("Failed to encode resized %s\n", f.Name())
			return
		}
	}

	fmt.Printf("Resizing %s\n", f.Name())

	if *flagLDPI {
		fn(ldpi)
	}
	fn(mdpi)
	fn(hdpi)
	fn(xhdpi)
	fn(xxhdpi)
	fn(xxxhdpi)

	return
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\n"+`Resizes resources by flag srcdpi to ldpi, mdpi,
hdpi, xhdpi, xxhdpi and xxxhdpi. srcdpi scaling is skipped by default. Specifying a
srcdpi below xxxhdpi will scale images up if -scaleup flag is set. Contents of flag path will
be walked and new files written to flag out will be flattened. This means
you can organize the contents of flag path with subdirectories to target
different interpolations for use on a particular set of images. For example:

	droidscale -path="xhdpi/L2" -interp="Lanzcos2"
	droidscale -path="xhdpi/L3" -interp="Lanzcos3"

Inspecting the contents of ./out/drawable-mdpi/ and friends will show a flat
list of the resized files.`)
}

func init() {
	flag.Usage = usage
	flag.Parse()
}

func main() {
	if *flagPath == "" {
		flag.Usage()
		return
	}

	switch *flagInterp {
	case "NearestNeighbor":
		interp = resize.NearestNeighbor
	case "Bilinear":
		interp = resize.Bilinear
	case "Bicubic":
		interp = resize.Bicubic
	case "MitchellNetravali":
		interp = resize.MitchellNetravali
	case "Lanzcos2":
		interp = resize.Lanczos2
	case "Lanzcos3":
		interp = resize.Lanczos3
	default:
		flag.Usage()
		fmt.Printf("\nReceived invalid interp: %s\nSee usage above.\n", *flagInterp)
		return
	}

	switch *flagSrcDpi {
	case "ldpi":
		srcdpi = ldpi
	case "mdpi":
		srcdpi = mdpi
	case "hdpi":
		srcdpi = hdpi
	case "xhdpi":
		srcdpi = xhdpi
	case "xxhdpi":
		srcdpi = xxhdpi
	case "xxxhdpi":
		srcdpi = xxxhdpi
	default:
		flag.Usage()
		fmt.Printf("\nReceived invalid srcdpi: %s\nSee usage above.\n", *flagSrcDpi)
		return
	}

	filepath.Walk(*flagPath, visit)
}
