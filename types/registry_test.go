package types

import "testing"

func fakeFactory(ProviderConfig) (Provider, error) { return nil, nil }

func TestRegisterDuplicateReturnsError(t *testing.T) {
	if err := Register("registry-test-dup", fakeFactory); err != nil {
		t.Fatalf("first Register should succeed, got: %v", err)
	}
	if err := Register("registry-test-dup", fakeFactory); err == nil {
		t.Fatal("second Register under the same name should return an error")
	}
}

func TestRegisterNilFactory(t *testing.T) {
	if err := Register("registry-test-nil", nil); err == nil {
		t.Fatal("Register with a nil factory should return an error")
	}
}

func TestLookup(t *testing.T) {
	if err := Register("registry-test-lookup", fakeFactory); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	f, ok := Lookup("registry-test-lookup")
	if !ok || f == nil {
		t.Fatal("Lookup should find a just-registered factory")
	}
	if _, ok := Lookup("registry-test-never-registered"); ok {
		t.Fatal("Lookup of an unregistered name should return ok=false")
	}
}

func TestRegisteredSorted(t *testing.T) {
	names := []string{"registry-test-zzz", "registry-test-aaa", "registry-test-mmm"}
	for _, n := range names {
		if err := Register(n, fakeFactory); err != nil {
			t.Fatalf("Register(%s) failed: %v", n, err)
		}
	}
	got := Registered()
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Fatalf("Registered() is not sorted: %v", got)
		}
	}
}
