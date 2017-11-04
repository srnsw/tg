package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	cdp "github.com/knq/chromedp"
	cdptypes "github.com/knq/chromedp/cdp"
	"github.com/knq/chromedp/client"
	"github.com/srnsw/tg"
)

func main() {
	ts := tg.Teams()
	for _, t := range ts {
		_, err := os.Stat(filepath.Join(tg.TGPATH, t.ID))
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(filepath.Join(tg.TGPATH, t.ID), 0777)
			}
			if err != nil {
				panic(err)
			}
		}
		err = scrape(t)
		if err == nil {
			var buf []byte
			buf, err = time.Now().GobEncode()
			if err == nil {
				err = ioutil.WriteFile(filepath.Join(tg.TGPATH, t.ID, "latest"), buf, 0644)
			}
		}
		if err != nil {
			log.Printf("error updating dashboard: %v", err)
		}
	}
}

// a non-headless mode for testing
func interactive(ctx context.Context, tasks cdp.Tasks) error {
	c, err := cdp.New(ctx, cdp.WithLog(log.Printf))
	if err != nil {
		return err
	}
	if err = c.Run(ctx, tasks); err != nil {
		return err
	}
	if err = c.Shutdown(ctx); err != nil {
		return err
	}
	return c.Wait()
}

func headless(ctx context.Context, tasks cdp.Tasks) error {
	c, err := cdp.New(ctx, cdp.WithTargets(client.New().WatchPageTargets(ctx)), cdp.WithLog(log.Printf))
	if err != nil {
		return err
	}
	return c.Run(ctx, tasks)
}

func scrape(t tg.Team) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tasks := cdp.Tasks{
		cdp.Navigate(`https://www.teamgage.com/Account/Login?ReturnUrl=%2FPortal%2F10077%2FReports%2FSingleReport%2F` + t.ID),
		cdp.Sleep(5 * time.Second),
		cdp.WaitVisible(`#Email`, cdp.ByID),
		cdp.SendKeys(`#Email`, t.User, cdp.ByID),
		cdp.WaitVisible(`#Password`, cdp.ByID),
		cdp.SendKeys(`#Password`, t.Pass, cdp.ByID),
		cdp.WaitVisible(`#login-btn`, cdp.ByID),
		cdp.Click(`#login-btn`, cdp.ByID),
		cdp.WaitVisible(`div.form-content`, cdp.ByQuery),
		cdp.Sleep(30 * time.Second),
	}
	byts := make([][]byte, 8)
	tasks = append(tasks, screenshots(byts)...)
	tasks = append(tasks, writes(t.ID, byts)...)
	return headless(ctx, tasks)
}

func screenshots(byts [][]byte) cdp.Tasks {
	tasks := make(cdp.Tasks, 0, 16)
	for i := range byts {
		tasks = append(tasks, cdp.ScrollIntoView("#content-header", cdp.ByID)) // scroll back to top before each screenshot - otherwise goes squew-iff
		sel := fmt.Sprintf("div.dashboard.masonry > div:nth-child(%d)", i+1)
		tasks = append(tasks, cdp.Screenshot(sel, &byts[i], cdp.ByQuery))
	}
	return tasks
}

func writes(id string, byts [][]byte) cdp.Tasks {
	return cdp.Tasks{cdp.ActionFunc(func(context.Context, cdptypes.Handler) error {
		byt, err := join(byts...)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(filepath.Join(tg.TGPATH, id, "dashboard.png"), byt, 0644)
	})}
}

func normalise(img image.Image) image.Image {
	rect := edges(img)
	rgba := image.NewRGBA(image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{rect.Max.X - rect.Min.X, rect.Max.Y - rect.Min.Y},
	})
	draw.Draw(rgba, rgba.Bounds(), img, rect.Min, draw.Src)
	return rgba
}

func edges(img image.Image) image.Rectangle {
	return image.Rectangle{min(img), max(img)}
}

func min(img image.Image) image.Point {
	white := color.RGBAModel.Convert(color.White)
	max := img.Bounds().Max
	for x := 0; x < max.X; x++ {
		for y := 0; y < max.Y; y++ {
			if img.At(x, y) != white {
				// now add a white border
				x, y = x-5, y-5
				if x < 0 {
					x = 0
				}
				if y < 0 {
					y = 0
				}
				return image.Pt(x, y)
			}
		}
	}
	return image.Point{}
}

func max(img image.Image) image.Point {
	white := color.RGBAModel.Convert(color.White)
	max := img.Bounds().Max
	for x := max.X - 1; x >= 0; x-- {
		for y := max.Y - 1; y >= 0; y-- {
			if img.At(x, y) != white {
				// now add a white border
				x, y = x+6, y+6
				if x > max.X {
					x = max.X
				}
				if y < max.Y {
					y = max.Y
				}
				return image.Pt(x, y)
			}
		}
	}
	return image.Point{}
}

// we tolerate slight differences in image dimensions
func difference(tolerance, a, b int) bool {
	if a-b > tolerance || b-a > tolerance {
		return true
	}
	return false
}

func join(bs ...[]byte) ([]byte, error) {
	if len(bs) > 8 || len(bs) < 1 {
		return nil, fmt.Errorf("expecting between 0 and 8 images, got %d", len(bs))
	}
	images := make([]image.Image, len(bs))
	var maxX, maxY int
	for i, v := range bs {
		buf := bytes.NewBuffer(v)
		img, err := png.Decode(buf)
		if err != nil {
			return nil, err
		}
		img = normalise(img)
		if maxX == 0 {
			maxX, maxY = img.Bounds().Max.X, img.Bounds().Max.Y
		} else {
			if difference(5, maxX, img.Bounds().Max.X) || difference(5, maxY, img.Bounds().Max.Y) {
				return nil, fmt.Errorf("expecting all images to have same dimensions (%d, %d): got %d, %d", maxX, maxY, img.Bounds().Max.X, img.Bounds().Max.Y)
			}
		}
		images[i] = img
	}
	rgba := image.NewRGBA(image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{maxX * 4, maxY * 2},
	})
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	for i, img := range images {
		var w, h int
		if i >= 4 {
			w = (i - 4) * maxX
			h = maxY
		} else {
			w = i * maxX
		}
		r := image.Rectangle{
			Min: image.Point{w, h},
			Max: image.Point{w + maxX, h + maxY},
		}
		draw.Draw(rgba, r, img, image.Point{0, 0}, draw.Src)
	}
	wr := &bytes.Buffer{}
	err := png.Encode(wr, rgba)
	return wr.Bytes(), err
}
