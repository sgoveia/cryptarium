package detector_test

import (
	"testing"

	"github.com/sgoveia/cryptarium/internal/detector"
	_ "github.com/sgoveia/cryptarium/internal/detector/certs"
)

func TestCertsRegisteredViaInit(t *testing.T) {
	found := false
	for _, d := range detector.All() {
		if d.Name() == "certs" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("certs detector not registered via init")
	}
}
