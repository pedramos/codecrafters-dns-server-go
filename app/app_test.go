package main

import (
	"os/exec"
	"testing"
)

// dig @127.0.0.1 -p 2053 +noedns codecrafters.io
func TestMain(t *testing.T) {
	appcmd := exec.Command("./app")
	if err := appcmd.Start(); err != nil {
		t.Fatal(err)
	}
	digcmd := exec.Command("dig", "@127.0.0.1", "-p", "2053", "+noedns", "+noall", "codecrafters.io/app")
	if err := digcmd.Run(); err != nil {
		t.Fatal(err)
	}

	err := appcmd.Wait()
	t.Errorf("Command finished with error: %v", err)

}
