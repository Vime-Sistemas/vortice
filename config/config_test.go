package config

import (
	"os"
	"reflect"
	"testing"
)

func TestGetLBAlgorithm(t *testing.T) {
	os.Setenv("LOAD_BALANCER_ALGO", "least_conn")
	defer os.Unsetenv("LOAD_BALANCER_ALGO")
	if GetLBAlgorithm() != "least_conn" {
		t.Fatalf("expected least_conn, got %s", GetLBAlgorithm())
	}
}

func TestGetIPHashHeader(t *testing.T) {
	os.Setenv("IP_HASH_HEADER", "X-Forwarded-For")
	defer os.Unsetenv("IP_HASH_HEADER")
	if GetIPHashHeader() != "X-Forwarded-For" {
		t.Fatalf("expected X-Forwarded-For, got %s", GetIPHashHeader())
	}
}

func TestGetPerBackendRateLimits(t *testing.T) {
	os.Setenv("RATE_LIMIT_RPS", "5")
	os.Setenv("RATE_LIMIT_BURST", "2")
	defer os.Unsetenv("RATE_LIMIT_RPS")
	defer os.Unsetenv("RATE_LIMIT_BURST")

	// no specific per-backend config: should return globals
	out := GetPerBackendRateLimits(3)
	expected := [][2]int{{5, 2}, {5, 2}, {5, 2}}
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("expected %v got %v", expected, out)
	}

	// specific per-backend config
	os.Setenv("BACKEND_RATE_LIMITS", "10/5,0/0,2/1")
	defer os.Unsetenv("BACKEND_RATE_LIMITS")
	out2 := GetPerBackendRateLimits(3)
	expected2 := [][2]int{{10, 5}, {0, 0}, {2, 1}}
	if !reflect.DeepEqual(out2, expected2) {
		t.Fatalf("expected %v got %v", expected2, out2)
	}
}
