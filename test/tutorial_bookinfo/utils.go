package tutorial_bookinfo

import "fmt"

func ready(e error) bool {
	if e != nil {
		return false
	}
	return true
}

func expectMatch(got, expected interface{}) error {
	if got == expected {
		return nil
	}
	return fmt.Errorf("got: %v, expected: %v", got, expected)
}
