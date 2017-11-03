package tg

import (
	"testing"
)

func TestUpdate(t *testing.T) {
	Register(Team{"8127", "richard", "sekret"})
	Register(Team{"817", "daniel", "baha"})
	Register(Team{"817", "daniel", "boo"})
	tea := Teams()
	if len(tea) != 2 || tea[1].Pass != "boo" {
		t.Fatal(tea)
	}
}

func TestDelete(t *testing.T) {
	Unregister(Team{"8127", "richard", "sekret"})
	Unregister(Team{"817", "daniel", "boo"})
	tea := Teams()
	if len(tea) != 0 {
		t.Fatal(tea)
	}
}
