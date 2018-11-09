package registry

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestFindEquivalent(t *testing.T) {
	eqv := EquivRegistries{
		Equivs: map[string][]string{
			"hello": {
				"bonjour",
				"hi",
			},
			"thanks": {
				"cheers",
			},
		},
	}
	if eqv.FindEquivalent("xxx") != "xxx" {
		t.Error("expected xxx -> xxx")
	}
	if eqv.FindEquivalent("hello") != "hello" {
		t.Error("expected hello -> hello")
	}
	if eqv.FindEquivalent("hi") != "hello" {
		t.Error("expected hi -> hello")
	}
	if eqv.FindEquivalent("cheers") != "thanks" {
		t.Error("expected cheers -> thanks")
	}
}

func createTempFile(t *testing.T) *os.File {
	file, err := ioutil.TempFile("", "equiv-regs.json")
	if err != nil {
		t.Fatal("failed to create temp file", err)
	}
	return file
}

func TestCreateEquivRegistries(t *testing.T) {
	t.Run("no file", func(t *testing.T) {
		eqr, err := CreateEquivRegistries("no-such-file")
		if err == nil || eqr != nil {
			t.Fatal("expected error and nil equiv registry")
		}
		if !strings.Contains(err.Error(), "no-such-file") {
			t.Errorf("expected no-such-file; got %s", err)
		}
	})
	t.Run("bad json", func(t *testing.T) {
		file := createTempFile(t)
		defer os.Remove(file.Name())
		content := []byte("rubbish")
		file.Write(content)
		eqr, err := CreateEquivRegistries(file.Name())
		if err == nil || eqr != nil {
			t.Fatal("expected error and nil equiv registry")
		}
		if !strings.Contains(err.Error(), "invalid character") {
			t.Errorf("expected invalid char error; got %s", err)
		}
	})
	t.Run("empty json", func(t *testing.T) {
		file := createTempFile(t)
		defer os.Remove(file.Name())
		content := []byte("{}")
		file.Write(content)
		eqr, err := CreateEquivRegistries(file.Name())
		if eqr == nil || err != nil {
			t.Fatalf("expected equiv registry and nil error; got %s", err)
		}
		if len(eqr.Equivs) != 0 {
			t.Errorf("expected no equivs; got %d", len(eqr.Equivs))
		}
	})
	t.Run("valid json", func(t *testing.T) {
		file := createTempFile(t)
		defer os.Remove(file.Name())
		content := []byte("{\"one\":[\"oneA\", \"oneB\"]}")
		file.Write(content)
		eqr, err := CreateEquivRegistries(file.Name())
		if eqr == nil || err != nil {
			t.Fatalf("expected equiv registry and nil error; got %s", err)
		}
		if len(eqr.Equivs) != 1 {
			t.Errorf("expected 1 equiv; got %d", len(eqr.Equivs))
		}
		if len(eqr.Equivs["one"]) != 2 {
			t.Errorf("expected 2 equiv entries; got %d", len(eqr.Equivs["one"]))
		}
	})
}
