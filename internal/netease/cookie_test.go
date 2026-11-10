package netease

import "testing"

// A cookie pasted from a browser comes in many shapes; the session is
// found in each, and nothing else is kept.
func TestParseCookie(t *testing.T) {
	cases := []struct {
		text      string
		music     string
		csrf      string
		noSession bool
	}{
		{text: "abc123", music: "abc123"},
		{text: "  abc123\n", music: "abc123"},
		{text: "MUSIC_U=abc123", music: "abc123"},
		{text: "NMTID=x; MUSIC_U=abc123; __csrf=ff; os=pc", music: "abc123", csrf: "ff"},
		{text: "Cookie: MUSIC_U=abc123;__csrf=ff", music: "abc123", csrf: "ff"},
		{text: `"MUSIC_U=abc123"`, music: "abc123"},
		{text: "MUSIC_U=\"abc123\"", music: "abc123"},
		{text: "", noSession: true},
		{text: "__csrf=ff; NMTID=x", noSession: true},
		{text: "MUSIC_U=", noSession: true},
		{text: "two words", noSession: true},
	}
	for _, tc := range cases {
		cookies := parseCookie(tc.text)
		if tc.noSession {
			if cookies != nil {
				t.Errorf("%q: found %v, want no session", tc.text, cookies)
			}
			continue
		}
		got := map[string]string{}
		for _, c := range cookies {
			got[c.Name] = c.Value
		}
		if got["MUSIC_U"] != tc.music || got["__csrf"] != tc.csrf || len(got) != len(cookies) {
			t.Errorf("%q: got %v", tc.text, got)
		}
		for name := range got {
			if name != "MUSIC_U" && name != "__csrf" {
				t.Errorf("%q: kept %s", tc.text, name)
			}
		}
	}
}
