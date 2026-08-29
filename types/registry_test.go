package types

import (
	"fmt"
	"sync"
	"testing"
)

func fakeFactory(ProviderConfig) (Provider, error) { return nil, nil }

func TestRegistryRegisterLookupDuplicate(t *testing.T) {
	t.Run("register succeeds", func(t *testing.T) {
		if err := Register("registry-test-register-succeeds", fakeFactory); err != nil {
			t.Fatalf("Register should succeed, got: %v", err)
		}
	})

	t.Run("lookup finds registered factory", func(t *testing.T) {
		if err := Register("registry-test-lookup-found", fakeFactory); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
		f, ok := Lookup("registry-test-lookup-found")
		if !ok || f == nil {
			t.Fatal("Lookup should find a just-registered factory")
		}
	})

	t.Run("lookup unregistered returns false", func(t *testing.T) {
		if _, ok := Lookup("registry-test-never-registered"); ok {
			t.Fatal("Lookup of an unregistered name should return ok=false")
		}
	})

	t.Run("duplicate register returns error", func(t *testing.T) {
		if err := Register("registry-test-dup", fakeFactory); err != nil {
			t.Fatalf("first Register should succeed, got: %v", err)
		}
		if err := Register("registry-test-dup", fakeFactory); err == nil {
			t.Fatal("second Register under the same name should return an error")
		}
	})
}

func TestRegisterNilFactory(t *testing.T) {
	if err := Register("registry-test-nil", nil); err == nil {
		t.Fatal("Register with a nil factory should return an error")
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

func TestRegistryConcurrentAccess(t *testing.T) {
	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("registry-test-concurrent-%d", i)
			if err := Register(name, fakeFactory); err != nil {
				t.Errorf("Register(%s) failed: %v", name, err)
			}
			if _, ok := Lookup(name); !ok {
				t.Errorf("Lookup(%s) should find the just-registered factory", name)
			}
			Registered()
		}(i)
	}
	wg.Wait()
}
