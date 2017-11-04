package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/srnsw/tg"
)

const serveAt = "localhost:5138"

func main() {
	http.HandleFunc("/team/", teamHandler)
	http.HandleFunc("/latest/", latestHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/unregister", unregisterHandler)
	http.HandleFunc("/ui", uiHandler)
	log.Fatal(http.ListenAndServe(serveAt, nil))
}

func teamHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/team/")
	if _, err := strconv.Atoi(id); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filepath.Join(tg.TGPATH, id, "dashboard.png"))
}

func latest(tid string) time.Time {
	var then time.Time
	buf, err := ioutil.ReadFile(filepath.Join(tg.TGPATH, tid, "latest"))
	if err == nil {
		(&then).GobDecode(buf)
	}
	return then
}

func latestHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/latest/")
	if _, err := strconv.Atoi(id); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, latest(id).Format(time.RFC822))
	return
}

func getTeam(r *http.Request) *tg.Team {
	id := r.FormValue("tgteam")
	if _, err := strconv.Atoi(id); err != nil {
		return nil
	}
	user, pass, token := r.FormValue("tguser"), r.FormValue("tgpass"), r.FormValue("tgtoken")
	if user == "" || pass == "" || !tg.Validate(token) {
		return nil
	}
	return &tg.Team{id, user, pass}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	t := getTeam(r)
	if t == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if tg.Register(*t) != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, "registration complete")
}

func unregisterHandler(w http.ResponseWriter, r *http.Request) {
	t := getTeam(r)
	if t == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if tg.Unregister(*t) != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, "successfully unregistered")
}

func uiHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `
	<html>
	<head><title>Teamgage scraper</title></head>
	<body>
		<h1>Teamgage scraper</h1>
		<div>
			<h2>Register a team</h2>
			<form action="/register" method="post">
			  Team:<br>
              <input type="text" name="tgteam"><br>
              User:<br>
              <input type="text" name="tguser"><br>
              Pass:<br>
              <input type="password" name="tgpass"><br>
              Token:<br>
              <input type="text" name="tgtoken"><br>
              <input type="submit">
			</form>
		</div>
		<div>
			<h2>Unregister a team</h2>
			<form action="/unregister" method="post">
			  Team:<br>
              <input type="text" name="tgteam"><br>
			  User:<br>
              <input type="text" name="tguser"><br>
              Pass:<br>
              <input type="password" name="tgpass"><br>
              Token:<br>
              <input type="text" name="tgtoken"><br>
              <input type="submit"><br>
			</form>
		</div>
	<body>
	</html>
	`)
}
