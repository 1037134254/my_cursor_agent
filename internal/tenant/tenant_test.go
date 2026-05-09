package tenant

import "testing"

func TestSanitizeID(t *testing.T) {
	if SanitizeID("acme_corp-1") != "acme_corp-1" {
		t.Fatal()
	}
	if SanitizeID("../x") != "" {
		t.Fatal()
	}
	if SanitizeID("") != "" {
		t.Fatal()
	}
}
