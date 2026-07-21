package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cc "github.com/jasielrt95/caribbean-cinemas-cli"
)

func TestBrowserOpenedMessage(t *testing.T) {
	m := tuiModel{screen: scrDetail}

	updated, _ := m.Update(browserOpenedMsg{})
	got := updated.(tuiModel)
	if !strings.Contains(got.status, "Opened the official") {
		t.Fatalf("status = %q, want browser-open confirmation", got.status)
	}

	updated, _ = m.Update(browserOpenedMsg{err: errors.New("launcher unavailable")})
	got = updated.(tuiModel)
	if got.err == nil || !strings.Contains(got.err.Error(), "launcher unavailable") {
		t.Fatalf("error = %v, want launcher failure", got.err)
	}
}

func TestOfficialCheckoutURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"showing":{"movie":{"id":"movie-1","name":"Moana","urlSlug":"moana","siteId":"45"}}}}`))
	}))
	defer server.Close()

	url, err := officialCheckoutURL(context.Background(), cc.New(cc.WithEndpoint(server.URL)), "showing-1")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://home.caribbeancinemas.com/plaza-americas/checkout/seats/showing-1"
	if url != want {
		t.Fatalf("URL = %q, want %q", url, want)
	}
}

func TestCheckoutRequiresShowing(t *testing.T) {
	cmd := checkoutCmd()
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--showing is required") {
		t.Fatalf("error = %v, want required showing", err)
	}
}

func TestBuildVersionIsNeverEmpty(t *testing.T) {
	if buildVersion() == "" {
		t.Fatal("build version must not be empty")
	}
}
