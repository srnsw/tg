package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	cdp "github.com/knq/chromedp"
	cdptypes "github.com/knq/chromedp/cdp"
)

var (
	team = os.Getenv("TGTEAM")
	user = os.Getenv("TGUSER")
	pass = os.Getenv("TGPASS")
	dir  = filepath.Join(os.TempDir(), "teamgage")
	then = time.Time{}
)

func main() {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0777)
		}
		if err != nil {
			panic(err)
		}
	}
	buf, err := ioutil.ReadFile(filepath.Join(dir, "latest"))
	if err == nil {
		(&then).GobDecode(buf)
	}
	http.HandleFunc("/tg", handler)
	http.ListenAndServe(":80", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if !(time.Now().Sub(then) < time.Hour*24) {
		err := scrape()
		if err == nil {
			var buf []byte
			then = time.Now()
			buf, err = then.GobEncode()
			if err == nil {
				err = ioutil.WriteFile(filepath.Join(dir, "latest"), buf, 0644)
			}
		}
		if err != nil {
			log.Printf("error updating dashboard: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.ServeFile(w, r, filepath.Join(dir, "dashboard.png"))
}

func scrape() error {
	ctxt, cancel := context.WithCancel(context.Background())
	defer cancel()
	// create chrome instance
	c, err := cdp.New(ctxt, cdp.WithLog(log.Printf))
	if err != nil {
		return err
	}
	tasks := cdp.Tasks{
		cdp.Navigate(`https://www.teamgage.com/Account/Login?ReturnUrl=%2fPortal%2f10077%2fReports%2fTeam%2f` + team),
		cdp.Sleep(5 * time.Second),
		cdp.WaitVisible(`#Email`, cdp.ByID),
		cdp.SendKeys(`#Email`, user, cdp.ByID),
		cdp.WaitVisible(`#Password`, cdp.ByID),
		cdp.SendKeys(`#Password`, pass, cdp.ByID),
		cdp.WaitVisible(`div.editor-submit > input.button-link`, cdp.ByQuery),
		cdp.Click(`div.editor-submit > input.button-link`, cdp.ByQuery),
		cdp.Sleep(2 * time.Second),
		cdp.WaitVisible(`div.footer`, cdp.ByQuery),
		cdp.WaitVisible(`div.form-content`, cdp.ByQuery),
	}
	byts := make([][]byte, 8)
	tasks = append(tasks, screenshots(byts)...)
	tasks = append(tasks, writes(byts)...)
	// run task list
	if err = c.Run(ctxt, tasks); err != nil {
		return err
	}
	// shutdown chrome
	err = c.Shutdown(ctxt)
	if err != nil {
		return err
	}
	// wait for chrome to finish
	return c.Wait()
}

func screenshots(byts [][]byte) cdp.Tasks {
	tasks := make(cdp.Tasks, 0, 16)
	for i := range byts {
		tasks = append(tasks, cdp.ScrollIntoView("div.header.dark-shadow", cdp.ByQuery)) // scroll back to top before each screenshot - otherwise goes squew-iff
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
		return ioutil.WriteFile(filepath.Join(dir, "dashboard.png"), byt, 0644)
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
