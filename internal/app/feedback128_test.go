package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/lyrics"
	"netease-music-mygo/internal/netease"
)

// Credits leave the lyrics shown once hidden, and come back when shown.
func TestShownLyricsHideCredits(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	lines := []lyrics.Line{
		{At: 0, Text: "作词 : 林夕"},
		{At: time.Second, Text: "作曲：陈小霞"},
		{At: 5 * time.Second, Text: "第一句"},
		{At: 9 * time.Second, Text: "制作人: 某某"},
		{At: 12 * time.Second, Text: "第二句"},
	}
	a.pl.SetLyrics(lines)
	if got, _ := a.shownLyrics(); len(got) != 5 {
		t.Fatalf("credits hidden by default: %d lines", len(got))
	}
	a.settings.HideCredits = true
	got, _ := a.shownLyrics()
	if len(got) != 2 || got[0].Text != "第一句" || got[1].Text != "第二句" {
		t.Fatalf("shown %v", got)
	}
	// The same lines are sifted once.
	if again, _ := a.shownLyrics(); &again[0] != &got[0] {
		t.Fatal("sifted again for the same lyrics")
	}
	// Lyrics of credits alone stay.
	a.pl.SetLyrics(lines[:2])
	if got, _ := a.shownLyrics(); len(got) != 2 {
		t.Fatalf("only credits: %d lines", len(got))
	}
}

// Ctrl+= and Ctrl+- zoom the window by tens, within bounds, and Ctrl+0
// zooms it back; the zoom is kept in the settings.
func TestZoomShortcuts(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	tt := ui.NewTester(a.view, 1000, 700)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyEqual)
	tt.Frame()
	if a.settings.UIZoom != 1.1 || a.uiZoom() != 1.1 {
		t.Fatalf("zoom %v after Ctrl+=", a.settings.UIZoom)
	}
	for range 20 {
		tt.Key(ui.Cmd, ui.KeyEqual)
	}
	if a.uiZoom() != maxUIZoom {
		t.Fatalf("zoom %v past the most", a.uiZoom())
	}
	for range 30 {
		tt.Key(ui.Cmd, ui.KeyMinus)
	}
	if a.uiZoom() != minUIZoom {
		t.Fatalf("zoom %v past the least", a.uiZoom())
	}
	tt.Key(ui.Cmd, ui.Key0)
	if a.settings.UIZoom != 0 || a.uiZoom() != 1 {
		t.Fatalf("zoom %v after Ctrl+0", a.settings.UIZoom)
	}
}

// Off, the line sung in Now Playing lights whole, as one with no words.
func TestNowPlayingWordsSwitch(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	l := lyrics.Line{Text: "谁知道这份渴望", Words: []lyrics.Word{{Text: "谁知道"}, {Text: "这份渴望"}}}
	if len(a.npLine(l).Words) != 2 {
		t.Fatal("not word by word by default")
	}
	a.settings.NpWholeLines = true
	if got := a.npLine(l); len(got.Words) != 0 || got.Text != l.Text {
		t.Fatalf("off: %+v", got)
	}
	if len(l.Words) != 2 {
		t.Fatal("the lyrics themselves lost their words")
	}
}

// The login dialog turns to the cookie field and back.
func TestLoginDialogCookieFace(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	a.login = loginState{open: true, gen: 1}
	tt := ui.NewTester(a.view, 1000, 700)
	tt.Advance(time.Second)
	if err := tt.Click("没有手机 App？使用 Cookie 登录"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !a.login.cookie || !tt.HasText("Cookie 登录") {
		t.Fatal("no cookie face")
	}
	if _, ok := tt.Find("Cookie"); !ok {
		t.Fatal("no cookie field")
	}
	if err := tt.Click("返回扫码登录"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if a.login.cookie || !tt.HasText("扫码登录") {
		t.Fatal("did not go back to the QR code")
	}
}

// In 热评模式 the hot comments show all along, through the singing too;
// only in the quiet parts, the singing keeps its lyrics.
func TestHotCommentsAllAlong(t *testing.T) {
	a, tr, lines := quietApp(0)
	if cm, _ := a.quietComment(&tr, lines, true, 22*time.Second); cm != nil {
		t.Fatalf("quiet parts only, showed %q in the singing", cm.Content)
	}
	a.settings.HotAll = true
	cm, next := a.quietComment(&tr, lines, true, 22*time.Second)
	if cm == nil || next <= 0 {
		t.Fatalf("all along, showed %v in the singing, next in %v", cm, next)
	}
	// They take turns, about one a slot.
	if later, _ := a.quietComment(&tr, lines, true, 22*time.Second+next); later == nil || later.ID == cm.ID {
		t.Fatalf("after %v, still %v", next, later)
	}
}

// 仅在间奏时显示热评 shows under 热评模式 once that is on, which turned on
// shows them all along.
func TestHotCommentsSettings(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	a.settingsView.open, a.settingsView.tab = true, settingsLyrics
	tt := ui.NewTester(a.view, 1100, 760)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Advance(time.Second)
	toggle := func(label, desc string) {
		t.Helper()
		title, ok1 := tt.Find(label)
		note, ok2 := tt.Find(desc)
		pane, ok3 := tt.Find("歌词设置项")
		if !ok1 || !ok2 || !ok3 {
			t.Fatalf("no %s control: %v %v %v", label, ok1, ok2, ok3)
		}
		tt.ClickAt(pane.X+pane.W-30-20-12, (title.Y+note.Y+note.H)/2)
		tt.Advance(time.Second)
	}
	if !tt.HasText("热评模式") || tt.HasText("仅在间奏时显示热评") {
		t.Fatal("want 热评模式 alone while off")
	}
	toggle("热评模式", "歌词处换成这首歌的热门评论，一条接一条地轮换")
	if !a.settings.HotComments || !a.settings.HotAll || !tt.HasText("仅在间奏时显示热评") {
		t.Fatalf("on: hot %v all %v", a.settings.HotComments, a.settings.HotAll)
	}
	toggle("仅在间奏时显示热评", "只在前奏、间奏与尾奏时显示热评，有歌词的地方照常显示歌词")
	if !a.settings.HotComments || a.settings.HotAll {
		t.Fatalf("quiet parts only: hot %v all %v", a.settings.HotComments, a.settings.HotAll)
	}
}

// The radars have a section of their own on the home page, a row that
// unfolds to all of them, each opening its playlist; 私人雷达 leaves the
// day's playlists above it.
func TestRadarSection(t *testing.T) {
	a := newTestApp()
	accentHome(a)
	a.profile = &netease.Profile{UserID: 1, Nickname: "听众"}
	a.recPlaylists[1] = netease.Playlist{ID: radars[0].id, Name: "私人雷达"}
	a.radar.lists = map[int64]netease.Playlist{radars[0].id: {ID: radars[0].id, Name: "今天从《喵》听起|私人雷达"}}
	tt := ui.NewTester(a.view, 1200, 1500)
	tt.Advance(time.Second)
	if !tt.HasText("今天从《喵》听起") || tt.HasText("韩系私人雷达") {
		t.Fatal("want a row of radars, the day's title under 私人雷达")
	}
	n := 0
	for _, s := range tt.Texts() {
		if s == "私人雷达" {
			n++
		}
	}
	if n != 3 { // the section's title, and its card's label and name
		t.Fatalf("私人雷达 shows %d times", n)
	}
	if err := tt.Click(fmt.Sprintf("展开全部 %d", len(radars))); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !a.radar.open || !tt.HasText("韩系私人雷达") {
		t.Fatal("did not unfold")
	}
	if err := tt.Click("时光雷达"); err != nil {
		t.Fatal(err)
	}
	if a.route.page != pagePlaylist || a.route.id != 5320167908 {
		t.Fatalf("route %+v", a.route)
	}
}

// 风格推荐 is a card two wide among the day's playlists: its first three
// songs each play from there, 换风格 opens the dialog, and the rest of it
// opens the songs' page.
func TestStyleCardPreview(t *testing.T) {
	a := newTestApp()
	accentHome(a)
	a.profile = &netease.Profile{UserID: 1, Nickname: "听众"}
	var songs []netease.Track
	for i, name := range []string{"Best Part", "Location", "Japanese Denim", "第四首"} {
		songs = append(songs, netease.Track{ID: int64(100 + i), Name: name, Duration: 200000, Artists: []netease.Artist{{Name: "歌手"}}, Album: netease.Album{PicURL: fmt.Sprintf("cover-%d", i+1)}})
	}
	a.style.songs = netease.StyleSongs{Songs: songs, Picked: netease.StyleCategory{ID: 1000, Tags: []netease.StyleTag{{ID: 1, Name: "R&B"}, {ID: 2, Name: "欧美流行"}}}}
	a.style.loaded = true
	a.style.cats = []netease.StyleCategory{{ID: 1000, Name: "曲风"}}
	tt := ui.NewTester(a.view, 1200, 1100)
	tt.Advance(time.Second)
	if dir := os.Getenv("NM_FRAMES"); dir != "" {
		f, _ := os.Create(filepath.Join(dir, "style-card.png"))
		png.Encode(f, tt.Image())
		f.Close()
	}
	if !tt.HasText("R&B · 欧美流行") || !tt.HasText("Japanese Denim") {
		t.Fatal("want the styles and the songs")
	}
	if err := tt.Click("播放 Location"); err != nil {
		t.Fatal(err)
	}
	if cur := a.pl.Current(); cur == nil || cur.ID != 101 || len(a.pl.Queue()) != 4 {
		t.Fatalf("playing %v", cur)
	}
	if a.route.page == pageStyle {
		t.Fatal("a song opened the page")
	}
	if err := tt.Click("换风格"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !a.style.open || a.route.page == pageStyle {
		t.Fatalf("换风格: open %v, page %v", a.style.open, a.route.page)
	}

	b := newTestApp()
	accentHome(b)
	b.profile, b.style = a.profile, styleState{songs: a.style.songs, loaded: true}
	tt = ui.NewTester(b.view, 1200, 1100)
	tt.Advance(time.Second)
	if err := tt.Click("R&B · 欧美流行"); err != nil {
		t.Fatal(err)
	}
	if b.route.page != pageStyle {
		t.Fatal("the card did not open the page")
	}
	// The page opens under its cover's colors, as a playlist's does.
	if got := b.pageArt(); got != "cover-1" {
		t.Fatalf("page art %q", got)
	}
}

// A song on the daily or the style card, once it plays, pauses and plays
// on from its row rather than starting again.
func TestCardPicksToggle(t *testing.T) {
	a := newTestApp()
	accentHome(a)
	a.profile = &netease.Profile{UserID: 1, Nickname: "听众"}
	var songs, day []netease.Track
	for i, name := range []string{"Best Part", "Location", "Japanese Denim", "第四首"} {
		songs = append(songs, netease.Track{ID: int64(100 + i), Name: name, Duration: 200000, Artists: []netease.Artist{{Name: "歌手"}}})
		day = append(day, netease.Track{ID: int64(200 + i), Name: "日" + name, Duration: 200000, Artists: []netease.Artist{{Name: "歌手"}}})
	}
	a.style.songs = netease.StyleSongs{Songs: songs, Picked: netease.StyleCategory{ID: 1000, Tags: []netease.StyleTag{{ID: 1, Name: "R&B"}}}}
	a.style.loaded = true
	a.daily = day
	tt := ui.NewTester(a.view, 1700, 1100)
	tt.Advance(time.Second)
	toggle := func(play, name string, id int64) {
		t.Helper()
		if err := tt.Click(play + name); err != nil {
			t.Fatal(err)
		}
		tt.Advance(time.Second)
		if cur := a.pl.Current(); cur == nil || cur.ID != id {
			t.Fatalf("%s: playing %v", name, cur)
		}
		was := a.pl.Playing()
		label := play + name
		if was {
			label = "暂停 " + name
		}
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Advance(time.Second)
		if cur := a.pl.Current(); cur == nil || cur.ID != id || a.pl.Playing() == was {
			t.Fatalf("%s: clicked again, playing %v %v", name, cur, a.pl.Playing())
		}
	}
	toggle("播放 ", "Location", 101)
	// The daily card's rows are named by their songs.
	toggle("", "日Best Part", 200)
	if dir := os.Getenv("NM_FRAMES"); dir != "" {
		r, _ := tt.Find("R&B")
		tt.Move(r.X+40, r.Y+60)
		tt.Advance(300 * time.Millisecond)
		f, _ := os.Create(filepath.Join(dir, "picks.png"))
		png.Encode(f, tt.Image())
		f.Close()
	}
}
