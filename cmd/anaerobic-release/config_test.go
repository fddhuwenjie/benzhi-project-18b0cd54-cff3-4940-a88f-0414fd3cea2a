package main

import "testing"

func TestValidateAddress(t *testing.T) {
	for _, valid := range []string{"127.0.0.1:19081", "[::1]:19082"} {
		if err := validateAddress(valid); err != nil {
			t.Errorf("%s: %v", valid, err)
		}
	}
	for _, invalid := range []string{"0.0.0.0:19081", "127.0.0.1:80", "localhost:19081", "bad"} {
		if err := validateAddress(invalid); err == nil {
			t.Errorf("%s 应被拒绝", invalid)
		}
	}
}
