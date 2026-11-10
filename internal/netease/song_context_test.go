package netease

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type songContextTransport func(*http.Request) (*http.Response, error)

func (f songContextTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Cancelling a superseded song must cancel the playback URL request itself.
func TestSongURLContextCancellation(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = songContextTransport(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SongURLContext(ctx, 1, "exhigh"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request returned %v", err)
	}
}

// An API outage or expired login must not make every song look unavailable.
func TestSongURLServiceErrorIsNotUnavailable(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for _, body := range []string{`{"code":301}`, `{"code":429}`, `{"code":500}`} {
		http.DefaultTransport = songContextTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		})
		if _, err := SongURL(1, "exhigh"); err == nil || errors.Is(err, ErrUnavailable) {
			t.Fatalf("service response %s returned %v", body, err)
		}
	}
}

// Empty or malformed responses do not establish that a song cannot play.
func TestSongURLIncompleteResponseIsRequestFailure(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for _, body := range []string{`{}`, `{"code":200}`, `{"code":200,"data":[]}`, `{"code":200,"data":[{"url":null}]}`, `<html>busy</html>`} {
		http.DefaultTransport = songContextTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		})
		var requestError *StreamError
		if _, err := SongURL(1, "exhigh"); !errors.As(err, &requestError) {
			t.Errorf("incomplete response %s classified as %T: %v", body, err, err)
		}
	}
}

// A usable URL remains usable when the response has no outer status code.
func TestSongURLAcceptsUsableDataWithoutOuterCode(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = songContextTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":[{"code":200,"url":"https://audio.test/song","type":"mp3"}]}`)), Request: r}, nil
	})
	st, err := SongURL(1, "exhigh")
	if err != nil || st.URL != "https://audio.test/song" {
		t.Fatalf("usable response failed: stream=%v error=%v", st, err)
	}
}

// An explicit preview still reports unavailability rather than reconnecting it forever.
func TestSongURLPreviewRemainsUnavailable(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for _, address := range []string{`null`, `"https://audio.test/preview"`} {
		http.DefaultTransport = songContextTransport(func(r *http.Request) (*http.Response, error) {
			body := `{"code":200,"data":[{"url":` + address + `,"freeTrialInfo":{"start":0,"end":30}}]}`
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		})
		if _, err := SongURL(1, "exhigh"); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("preview was classified as %v", err)
		}
	}
}

// When the desktop API answers -460, the web player's stream is used.
func TestSongURLFallsBackToWeb(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = songContextTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"code":-460}`
		if r.URL.Host == "music.163.com" {
			body = `{"code":200,"data":[{"url":"http://audio.test/web","type":"mp3","size":123,"level":"exhigh","peak":0.5}]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	st, err := SongURL(1, "exhigh")
	if err != nil {
		t.Fatal(err)
	}
	if st.URL != "https://audio.test/web" || st.Size != 123 || st.Type != "mp3" || st.Peak != 0.5 {
		t.Fatalf("stream %+v", st)
	}
}

// A song's own 404 means no rights to it, and -110 no right to play it:
// both skip, rather than retry.
func TestSongURLItemCodesAreUnavailable(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for code, want := range map[string]error{"404": ErrNoCopyright, "-110": ErrUnavailable} {
		http.DefaultTransport = songContextTransport(func(r *http.Request) (*http.Response, error) {
			body := `{"code":200,"data":[{"id":1,"url":null,"code":` + code + `}]}`
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		})
		if _, err := SongURL(1, "exhigh"); !errors.Is(err, want) {
			t.Errorf("item code %s returned %v", code, err)
		}
	}
	if !errors.Is(ErrNoCopyright, ErrUnavailable) {
		t.Fatal("no copyright is not unavailable")
	}
}
