package main

import (
	"os/exec"
	"testing"
)

func TestParse(t *testing.T) {
	if testing.Short() {
		t.Skip("this test is slow")
	}

	// parse last 10 commits
	//cmd := exec.Command("git", "fast-export", "HEAD~10...HEAD")
	cmd := exec.Command("git", "fast-export", "HEAD")
	//cmd.Dir = "/Users/keegan/go/src/github.com/keegancsmith/sqlf"
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	err = parse(out)
	if err != nil {
		t.Fatal(err)
	}

	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
