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
	"strconv"
	"strings"
	"time"

	cdp "github.com/knq/chromedp"
	cdptypes "github.com/knq/chromedp/cdp"
	"github.com/knq/chromedp/client"
)

var tgpath = os.Getenv("TGPATH")

func main() {
	if tgpath == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		tgpath = filepath.Join(u.HomeDir, "teamgage")
	}
	http.HandleFunc("/team/", handler)
	log.Fatal(http.ListenAndServe("localhost:5138", nil)) // :80
}

func latest(tid string) time.Time {
	var then time.Time
	buf, err := ioutil.ReadFile(filepath.Join(tgpath, tid, "latest"))
	if err == nil {
		(&then).GobDecode(buf)
	}
	return then
}

type team struct {
	id   string
	user string
	pass string
}

func getTeam(r *http.Request) *team {
	if r.Method != "POST" {
		return nil
	}
	id := strings.TrimPrefix(r.URL.Path, "/team/")
	if _, err := strconv.Atoi(id); err != nil {
		return nil
	}
	_, err := os.Stat(filepath.Join(tgpath, id))
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(filepath.Join(tgpath, id), 0777)
		}
		if err != nil {
			panic(err)
		}
	}
	user, pass := r.FormValue("tguser"), r.FormValue("tgpass")
	if user == "" || pass == "" {
		return nil
	}
	return &team{id, user, pass}
}

func handler(w http.ResponseWriter, r *http.Request) {
	t := getTeam(r)
	if t == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if !(time.Now().Sub(latest(t.id)) < time.Hour*24) {
		err := scrape(t)
		if err == nil {
			var buf []byte
			buf, err = time.Now().GobEncode()
			if err == nil {
				err = ioutil.WriteFile(filepath.Join(tgpath, t.id, "latest"), buf, 0644)
			}
		}
		if err != nil {
			log.Printf("error updating dashboard: %v", err)
		}
	}
	http.ServeFile(w, r, filepath.Join(tgpath, t.id, "dashboard.png"))
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

func scrape(t *team) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tasks := cdp.Tasks{
		cdp.Navigate(`https://www.teamgage.com/Account/Login?ReturnUrl=%2FPortal%2F10077%2FReports%2FSingleReport%2F` + t.id),
		cdp.Sleep(5 * time.Second),
		cdp.WaitVisible(`#Email`, cdp.ByID),
		cdp.SendKeys(`#Email`, t.user, cdp.ByID),
		cdp.WaitVisible(`#Password`, cdp.ByID),
		cdp.SendKeys(`#Password`, t.pass, cdp.ByID),
		cdp.WaitVisible(`#login-btn`, cdp.ByID),
		cdp.Click(`#login-btn`, cdp.ByID),
		cdp.WaitVisible(`div.form-content`, cdp.ByQuery),
		cdp.Sleep(30 * time.Second),
	}
	byts := make([][]byte, 8)
	tasks = append(tasks, screenshots(byts)...)
	tasks = append(tasks, writes(t.id, byts)...)
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
		byt, err := join(id, byts...)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(filepath.Join(tgpath, id, "dashboard.png"), byt, 0644)
	})}
}

func join(id string, bs ...[]byte) ([]byte, error) {
	if len(bs) > 8 {
		return nil, fmt.Errorf("expecting 8 images or less, got %d", len(bs))
	}
	rgba := image.NewRGBA(image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{206 * 4, 356 * 2},
	})
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	for i, v := range bs {
		ioutil.WriteFile(filepath.Join(tgpath, id, strconv.Itoa(i)+".png"), v, 0644)
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
		draw.Draw(rgba, r, img, image.ZP, draw.Src)
	}
	wr := &bytes.Buffer{}
	err := png.Encode(wr, rgba)
	return wr.Bytes(), err
}
