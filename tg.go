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
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"

	cdp "github.com/knq/chromedp"
	cdptypes "github.com/knq/chromedp/cdp"
	"github.com/knq/chromedp/client"
)

var then time.Time

var (
	tgteam = os.Getenv("TGTEAM")
	tguser = os.Getenv("TGUSER")
	tgpass = os.Getenv("TGPASS")
	tgpath = os.Getenv("TGPATH")
)

func main() {
	if tgpath == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		tgpath = filepath.Join(u.HomeDir, "teamgage")
	}
	_, err := os.Stat(tgpath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(tgpath, 0777)
		}
		if err != nil {
			panic(err)
		}
	}
	buf, err := ioutil.ReadFile(filepath.Join(tgpath, "latest"))
	if err == nil {
		(&then).GobDecode(buf)
	}
	http.HandleFunc("/tg", handler)
	log.Fatal(http.ListenAndServe(":80", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if !(time.Now().Sub(then) < time.Hour*24) {
		err := scrape()
		if err == nil {
			var buf []byte
			then = time.Now()
			buf, err = then.GobEncode()
			if err == nil {
				err = ioutil.WriteFile(filepath.Join(tgpath, "latest"), buf, 0644)
			}
		}
		if err != nil {
			log.Printf("error updating dashboard: %v", err)
		}
	}
	http.ServeFile(w, r, filepath.Join(tgpath, "dashboard.png"))
}

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

func scrape() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tasks := cdp.Tasks{
		cdp.Navigate(`https://www.teamgage.com/Account/Login?ReturnUrl=%2FPortal%2F10077%2FReports%2FSingleReport%2F` + tgteam),
		cdp.Sleep(5 * time.Second),
		cdp.WaitVisible(`#Email`, cdp.ByID),
		cdp.SendKeys(`#Email`, tguser, cdp.ByID),
		cdp.WaitVisible(`#Password`, cdp.ByID),
		cdp.SendKeys(`#Password`, tgpass, cdp.ByID),
		cdp.WaitVisible(`#login-btn`, cdp.ByID),
		cdp.Click(`#login-btn`, cdp.ByID),
		cdp.WaitVisible(`div.form-content`, cdp.ByQuery),
		cdp.Sleep(30 * time.Second),
	}
	byts := make([][]byte, 8)
	tasks = append(tasks, screenshots(byts)...)
	tasks = append(tasks, writes(byts)...)
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

func writes(byts [][]byte) cdp.Tasks {
	return cdp.Tasks{cdp.ActionFunc(func(context.Context, cdptypes.Handler) error {
		byt, err := join(byts...)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(filepath.Join(tgpath, "dashboard.png"), byt, 0644)
	})}
}

func join(bs ...[]byte) ([]byte, error) {
	if len(bs) > 8 {
		return nil, fmt.Errorf("expecting 8 images or less, got %d", len(bs))
	}
	rgba := image.NewRGBA(image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{206 * 4, 356 * 2},
	})
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	for i, v := range bs {
		buf := bytes.NewBuffer(v)
		img, err := png.Decode(buf)
		if err != nil {
			return nil, err
		}
		var w, h int
		if i >= 4 {
			w = (i - 4) * 206
			h = 356
		} else {
			w = i * 206
		}
		r := image.Rectangle{
			Min: image.Point{w, h},
			Max: image.Point{w + 206, h + 356},
		}
		draw.Draw(rgba, r, img, image.Point{0, 0}, draw.Src)
	}
	wr := &bytes.Buffer{}
	err := png.Encode(wr, rgba)
	return wr.Bytes(), err
}
