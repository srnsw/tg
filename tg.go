package tg

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
)

var (
	TGPATH  = os.Getenv("TGPATH")
	tgtoken = os.Getenv("TGTOKEN")

	pathErr = errors.New("TGPATH isn't set")
)

func init() {
	if TGPATH == "" {
		u, err := user.Current()
		if err != nil {
			return
		}
		TGPATH = filepath.Join(u.HomeDir, "teamgage")
	}
	_, err := os.Stat(TGPATH)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(TGPATH, 0777)
		}
		if err != nil {
			return
		}
	}
}

type Team struct {
	ID   string
	User string
	Pass string
}

func Validate(tok string) bool {
	return tok == tgtoken
}

func Register(nt Team) error {
	if TGPATH == "" {
		return pathErr
	}
	teams := Teams()
	var exists bool
	for i, t := range teams {
		if t.ID == nt.ID {
			if t.User == nt.User && t.Pass == nt.Pass {
				return nil
			}
			teams[i] = nt
			exists = true
			break
		}
	}
	if !exists {
		teams = append(teams, nt)
	}
	byts, err := json.Marshal(teams)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(TGPATH, "teams.json"), byts, 0644)
}

func Unregister(nt Team) error {
	if TGPATH == "" {
		return pathErr
	}
	teams := Teams()
	var exists bool
	var tid int
	for i, t := range teams {
		if t.ID == nt.ID {
			if t.User == nt.User && t.Pass == nt.Pass {
				exists = true
				tid = i
			}
			break
		}
	}
	if !exists {
		return errors.New("bad ID or credentials")
	}
	teams = append(teams[:tid], teams[tid+1:]...)
	byts, err := json.Marshal(teams)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(TGPATH, "teams.json"), byts, 0644)
}

func Teams() []Team {
	var t []Team
	byts, err := ioutil.ReadFile(filepath.Join(TGPATH, "teams.json"))
	if err != nil {
		return t
	}
	json.Unmarshal(byts, &t)
	return t
}
