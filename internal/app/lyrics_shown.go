package app

import (
	"netease-music-mygo/internal/lyrics"
)

// shownCache is the song's lyrics without their credits, kept while the
// player holds the same lines, not to sift them every frame.
type shownCache struct {
	from  *lyrics.Line
	n     int
	lines []lyrics.Line
}

// shownLyrics are the lyrics of the song playing as Now Playing and the
// desktop lyrics show them: without the lines naming who made it, as 作词
// and 作曲, if the listener hid them.
func (a *app) shownLyrics() ([]lyrics.Line, bool) {
	lines, loaded := a.pl.Lyrics()
	if !a.settings.HideCredits || len(lines) == 0 {
		return lines, loaded
	}
	sc := &a.shownLines
	if sc.from != &lines[0] || sc.n != len(lines) {
		sc.from, sc.n = &lines[0], len(lines)
		sc.lines = withoutCredits(lines)
	}
	return sc.lines, loaded
}

// withoutCredits returns lines without the credits among them. Lyrics that
// are only credits are left as they are: better than none.
func withoutCredits(lines []lyrics.Line) []lyrics.Line {
	out := make([]lyrics.Line, 0, len(lines))
	for _, l := range lines {
		if !lyrics.IsCredit(l.Text) {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return lines
	}
	return out
}

// npLine is l as Now Playing lights it: word by word, or, with the
// switch off, whole at once, as a line with no words.
func (a *app) npLine(l lyrics.Line) lyrics.Line {
	if a.settings.NpWholeLines {
		l.Words = nil
	}
	return l
}
