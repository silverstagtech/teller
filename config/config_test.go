package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	c, err := New("./example.json")
	if err != nil {
		t.Logf("Failed to create a config. Errors %s", err)
		t.Fail()
	}
	t.Log(c)
}
