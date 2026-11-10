package player

import (
	"context"
	"errors"
	"testing"
	"time"

	"netease-music-mygo/internal/netease"
)

func gaplessPlayer(t *testing.T) (*Player, chan func()) {
	t.Helper()
	quietNavigation(t)
	oldCover := fetchCover
	fetchCover = func(netease.Track) ([]byte, error) { return nil, errors.New("no cover") }
	t.Cleanup(func() { fetchCover = oldCover })
	songURL = func(context.Context, int64, string) (*netease.Stream, error) {
		return nil, errors.New("songs come from the cache")
	}
	p, ch := navigationPlayer(t, []netease.Track{{ID: 1, Duration: 60000}, {ID: 2, Duration: 60000}})
	cacheNavigationTrack(t, p, 1)
	cacheNavigationTrack(t, p, 2)
	// What closes later — a preload, a tail — closes before the cache goes.
	t.Cleanup(func() {
		if n := p.pre; n != nil {
			p.pre = nil
			n.discard()
		}
		p.cutTail()
		for end := time.Now().Add(400 * time.Millisecond); time.Now().Before(end); {
			select {
			case fn := <-ch:
				fn()
			case <-time.After(20 * time.Millisecond):
			}
		}
	})
	p.load(0)
	finishNavigation(t, p, ch)
	if p.out == nil {
		t.Fatalf("first track did not load: %s", p.err)
	}
	return p, ch
}

// waitPreload runs what the player posts until the next track's audio is in.
func waitPreload(t *testing.T, p *Player, ch chan func()) {
	t.Helper()
	if p.pre == nil {
		t.Fatal("nothing preloads")
	}
	select {
	case <-p.pre.audio:
	case <-time.After(3 * time.Second):
		t.Fatal("preload did not finish")
	}
	if p.pre.err != nil {
		t.Fatal(p.pre.err)
	}
}

// Near its end, a track loads the next, which then starts the moment it
// is asked for, without waiting for anything to load.
func TestGaplessNextStartsAtOnce(t *testing.T) {
	p, ch := gaplessPlayer(t)
	p.nearEnd(10 * time.Second)
	waitPreload(t, p, ch)
	p.Next()
	if p.Current().ID != 2 || p.out == nil || p.Loading() {
		t.Fatalf("next did not start at once: id=%d out=%v loading=%v", p.Current().ID, p.out != nil, p.Loading())
	}
	if p.pre != nil {
		t.Fatal("the preload was not taken")
	}
}

// A preload of a track no longer next is dropped, not played.
func TestGaplessStalePreloadDropped(t *testing.T) {
	p, ch := gaplessPlayer(t)
	p.nearEnd(10 * time.Second)
	waitPreload(t, p, ch)
	p.Jump(0)
	if p.pre != nil {
		t.Fatal("a preload of another track was kept")
	}
	finishNavigation(t, p, ch)
	if p.Current().ID != 1 || p.out == nil {
		t.Fatalf("jump played %d", p.Current().ID)
	}
}

// With crossfade on, the next song starts silent and rises while the one
// ending fades out under it.
func TestCrossfadeOverlaps(t *testing.T) {
	p, ch := gaplessPlayer(t)
	p.Crossfade = func() time.Duration { return 3 * time.Second }
	p.nearEnd(10 * time.Second)
	waitPreload(t, p, ch)
	p.nearEnd(2 * time.Second)
	if p.Current().ID != 2 || p.out == nil {
		t.Fatalf("crossfade did not move on: id=%d", p.Current().ID)
	}
	if p.tail == nil || p.tail.out == nil {
		t.Fatal("the song before does not fade out under the next")
	}
	if lvl := p.fade.level(time.Now()); lvl > 0.1 {
		t.Fatalf("next song starts at level %v, not faded in", lvl)
	}
	// Pausing cuts the tail.
	p.TogglePlay()
	if p.tail != nil {
		t.Fatal("pause left the tail playing")
	}
}

// With crossfade on but the next song not in yet, this one fades out as it
// ends, and seeking back brings it up again.
func TestCrossfadeFallbackAndSeekBack(t *testing.T) {
	p, _ := gaplessPlayer(t)
	p.Crossfade = func() time.Duration { return 3 * time.Second }
	p.nearEnd(2 * time.Second)
	if p.Current().ID != 1 || p.fadeInNext != 3*time.Second || p.endFade != p.gen {
		t.Fatalf("no fade out: id=%d fadeInNext=%v", p.Current().ID, p.fadeInNext)
	}
	p.Seek(5 * time.Second)
	if p.fadeInNext != 0 || p.endFade == p.gen {
		t.Fatal("seeking back kept the end's fade")
	}
	if p.fade.to != 1 {
		t.Fatalf("seeking back fades to %v", p.fade.to)
	}
}
