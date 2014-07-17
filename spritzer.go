package main

import (
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"text/template"
)

func main() {
	errHandle(spritzer)
	fmt.Println("Success <3 ")
}

func errHandle(f func() error) {
	if err := f(); err != nil {
		fmt.Println("error: ", err)
	}
	return
}

func spritzer() error {
	s := sprite{}

	files, err := ioutil.ReadDir("./")
	if err != nil {
		return err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".png") && !strings.Contains(f.Name(), "Retina") && !strings.Contains(f.Name(), "sprite") {
			reader, err := os.Open(f.Name())
			if err != nil {
				return err
			}

			imgConfig, err := png.DecodeConfig(reader)
			if err != nil {
				return err
			}
			reader.Close()

			s.images = append(s.images, img{strings.TrimSuffix(f.Name(), ".png"), image.Rect(0, 0, imgConfig.Width, imgConfig.Height)})
		}
	}

	sort.Sort(byHeight(s.images))

	var sumX int
	for i := 0; i < len(s.images); i++ {
		s.images[i].Box = s.images[i].Box.Add(image.Point{sumX, 0})
		sumX += s.images[i].Box.Dx()
	}

	// Check for images
	if s.images == nil {
		err = fmt.Errorf("no images")
		return err
	}

	s.ImagesOpt = append([]img{}, s.images...)
	s.BaseOpt = image.Point{sumX, s.images[0].Box.Dy()}

	// Iteration
	var sumY int
	imgNo := len(s.images)
	for j := 0; j < imgNo; j++ {
		sumY += s.images[j].Box.Dy()
		b := boxes{inf: []image.Rectangle{image.Rect(0, 0, 1, sumY)}}
		for i := 0; i < imgNo; i++ {
			s.images[i].Box = b.boxFind(s.images[i].Box)
			b.boxCut(s.images[i].Box)
		}
		s.base = image.Point{0, 0}
		for i := 0; i < imgNo; i++ {
			if s.images[i].Box.Max.X > s.base.X {
				s.base = image.Point{s.images[i].Box.Max.X, sumY}
			}
		}
		if s.base.X*s.base.Y < s.BaseOpt.X*s.BaseOpt.Y {
			s.ImagesOpt = append([]img{}, s.images...)
			s.BaseOpt = image.Point{s.base.X, s.base.Y}
		}
	}

	// Write sprite, retina support
	m := image.NewRGBA(image.Rect(0, 0, s.BaseOpt.X, s.BaseOpt.Y))
	mRetina := image.NewRGBA(image.Rect(0, 0, s.BaseOpt.X*2, s.BaseOpt.Y*2))
	draw.Draw(m, m.Bounds(), image.Transparent, image.ZP, draw.Src)
	draw.Draw(mRetina, mRetina.Bounds(), image.Transparent, image.ZP, draw.Src)

	var iRetina image.Image
	for _, f := range s.ImagesOpt {
		i := pngDecode(f.Name + ".png")

		// Check for retina file
		if _, err := os.Stat(f.Name + "Retina.png"); err == nil {
			iRetina = pngDecode(f.Name + "Retina.png")
		} else {
			// Resize image !should not be used
			fmt.Println("warning: non retina image being used: ", f.Name+".png")
			iRetina = resize.Resize(uint(f.Box.Dx()*2), 0, i, resize.Lanczos3)
		}

		fBoxRetina := image.Rect(f.Box.Min.X*2, f.Box.Min.Y*2, f.Box.Max.X*2, f.Box.Max.Y*2)

		draw.Draw(m, f.Box, i, image.Point{0, 0}, draw.Src)
		draw.Draw(mRetina, fBoxRetina, iRetina, image.Point{0, 0}, draw.Src)
	}

	w, err := os.Create("sprite.png")
	if err != nil {
		return err
	}
	wRetina, err := os.Create("spriteRetina.png")
	if err != nil {
		return err
	}

	png.Encode(w, m)
	w.Close()
	png.Encode(wRetina, mRetina)
	wRetina.Close()

	// Encode css
	// .spriteName {background-position: widthpx heightpx}
	w, err = os.Create("sprite.css")
	if err != nil {
		return err
	}

	tmp := `
.sprite {
  background-image:url("../img/sprite.png") !important;
  background-size: {{.BaseOpt.X}}px {{.BaseOpt.Y}}px;
  background-repeat: no-repeat;
  background-color: transparent;
  overflow: hidden;
}

@media 
(-webkit-min-device-pixel-ratio: 2), (min-resolution: 192dpi) { 
  .sprite {
    background-image:url("../img/spriteRetina.png") !important;
    background-size: {{.BaseOpt.X}}px {{.BaseOpt.Y}}px;
  }
}

{{range $a := .ImagesOpt}}.sprite{{$a.Name}} {background-position: -{{$a.Box.Min.X}}px -{{$a.Box.Min.Y}}px}` + "\n" + `{{end}}`

	tmpl, err := template.New("css").Parse(tmp)
	if err != nil {
		return err
	}
	err = tmpl.Execute(w, s)
	if err != nil {
		return err
	}
	w.Close()

	return nil
}

type sprite struct {
	images, ImagesOpt []img
	base, BaseOpt     image.Point
}

type img struct {
	Name string
	Box  image.Rectangle
}

type boxes struct {
	inf, box []image.Rectangle
}

func (b *boxes) boxFind(i image.Rectangle) image.Rectangle {
	var min image.Point
	flag := true
	for _, box := range b.inf {
		if box.Dy() >= i.Dy() {
			if flag {
				min = box.Min
				flag = false
			} else if min.X > box.Min.X {
				min = box.Min
			}
		}
	}
	/*
		for _, box := range b.box {
			if box.Dx() > i.Dx() && box.Dy() >= i.Dy() {
				if min.X > box.Min.X {
					min = box.Min
				}
			}
		}
	*/
	return image.Rectangle{min, min.Add(i.Size())}
}

func (b *boxes) boxCut(i image.Rectangle) {
	iy0, ix1, iy1 := i.Min.Y, i.Max.X, i.Max.Y

	loop := len(b.inf)
	del := []int{}
	for n := 0; n < loop; n++ {
		bx0, by0, by1 := b.inf[n].Min.X, b.inf[n].Min.Y, b.inf[n].Max.Y
		// Determine if new box required
		if ((iy0 >= by0 && iy0 < by1) || (iy1 > by0 && iy1 <= by1)) && ix1 > bx0 {
			b.newBoxInf(bx0, iy1, bx0+1, by1)
			b.newBoxInf(ix1, by0, ix1+1, by1)
			b.newBoxInf(bx0, by0, bx0+1, iy0)
			// Mark for deletion
			del = appendIfUnique(del, n)
		}
	}
	b.delete(del)
	del = []int{}
	// Garbage collection
	for n, a := range b.inf {
		for m, b := range b.inf {
			if a.In(b) && n != m {
				del = appendIfUnique(del, n)
			}
		}
	}
	b.delete(del)

	return
}

func (b *boxes) delete(array []int) {
	s := 0
	for _, n := range array {
		n = n + s
		b.inf = append(b.inf[:n], b.inf[n+1:]...)
		s = s - 1
	}
}

func appendIfUnique(slice []int, i int) []int {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func (b *boxes) newBox(x0, y0, x1, y1 int) {
	if x0 > x1 || y0 > y1 {
		return
	}
	box := image.Rectangle{image.Point{x0, y0}, image.Point{x1, y1}}
	if !box.Empty() {
		b.box = append(b.box, box)
	}
	return
}

func (b *boxes) newBoxInf(x0, y0, x1, y1 int) {
	if x0 > x1 || y0 > y1 {
		return
	}
	box := image.Rectangle{image.Point{x0, y0}, image.Point{x1, y1}}
	if !box.Empty() {
		b.inf = append(b.inf, box)
	}
	return
}

func pngDecode(file string) image.Image {
	reader, err := os.Open(file)
	if err != nil {
		panic(err)
	}

	i, err := png.Decode(reader)
	if err != nil {
		panic(err)
	}
	reader.Close()
	return i
}

// byHeight implements the sort.Interface for []img based on hieght field
type byHeight []img

func (a byHeight) Len() int           { return len(a) }
func (a byHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byHeight) Less(i, j int) bool { return a[i].Box.Dy() > a[j].Box.Dy() }
