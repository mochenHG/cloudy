package app

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// NetEase makes more radars than the 私人雷达 the day's playlists hold: one
// of old songs, of new ones, of hidden gems, one in each language… each a
// playlist it fills anew every day for whoever is signed in. The home page
// gives them a section of their own under 今日为你推荐, a row of them
// that unfolds to all.

var iconRadar = strokeIcon(`<path d="M19.07 4.93A10 10 0 0 0 6.99 3.34"/><path d="M4 6h.01"/><path d="M2.29 9.62A10 10 0 1 0 21.31 8.35"/><path d="M16.24 7.76A6 6 0 1 0 8.23 16.67"/><path d="M12 18h.01"/><path d="M17.99 11.66A6 6 0 0 1 15.77 16.67"/><circle cx="12" cy="12" r="2"/><path d="m13.41 10.59 5.66-5.66"/>`)

// radar is one of NetEase's radar playlists.
type radar struct {
	id   int64
	name string
	desc string
}

// radars are NetEase's radars, 私人雷达 first. Their ids are the same for
// everyone; their songs are picked for whoever opens them.
var radars = []radar{
	{3136952023, "私人雷达", "按你的口味挑选"},
	{5320167908, "时光雷达", "重温听过的老歌"},
	{5362359247, "宝藏雷达", "小众好歌"},
	{5300458264, "新歌雷达", "常听歌手的新歌"},
	{5327906368, "乐迷雷达", "常听歌手的更多作品"},
	{5341776086, "神秘雷达", "意想不到的好歌"},
	{8819359201, "热点雷达", "正在流行的歌"},
	{8402996200, "会员雷达", "黑胶会员专属推荐"},
	{2829883282, "华语私人雷达", "最懂你的华语推荐"},
	{2829816518, "欧美私人雷达", "最懂你的欧美推荐"},
	{2829896389, "日系私人雷达", "最懂你的日系推荐"},
	{2829920189, "韩系私人雷达", "最懂你的韩系推荐"},
}

// isRadar reports whether id is one of the radars.
func isRadar(id int64) bool {
	for _, r := range radars {
		if r.id == id {
			return true
		}
	}
	return false
}

// radarState is what the home page shows of the radars: their covers and
// the day's titles, as NetEase made them for the listener, and whether the
// section is unfolded.
type radarState struct {
	lists   map[int64]netease.Playlist
	loading bool
	open    bool
}

var playlistSummary = netease.PlaylistSummary

// loadRadars loads the radars' covers and titles for whoever is signed in.
func (a *app) loadRadars() {
	if a.profile == nil {
		a.radar = radarState{open: a.radar.open}
		return
	}
	if a.radar.loading {
		return
	}
	a.radar.loading = true
	gen := a.accountGen
	go func() {
		lists := map[int64]netease.Playlist{}
		var mu sync.Mutex
		var wg sync.WaitGroup
		for _, r := range radars {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if pl, err := playlistSummary(r.id); err == nil && pl != nil {
					mu.Lock()
					lists[r.id] = *pl
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		a.post(func() {
			if gen != a.accountGen {
				return
			}
			a.radar.loading, a.radar.lists = false, lists
		})
	}()
}

// radarTitle is the day's title NetEase gave a radar, as "今天从《喵》听起"
// of "今天从《喵》听起|私人雷达", or what the radar is about.
func radarTitle(r radar, pl netease.Playlist) string {
	if t, _, ok := strings.Cut(pl.Name, "|"); ok && strings.TrimSpace(t) != "" {
		return strings.TrimSpace(t)
	}
	return r.desc
}

// radarSection is the radars' section: a row of them, unfolding to all.
func (a *app) radarSection(c *ui.Context) {
	if a.profile == nil {
		return
	}
	p := a.pal
	cols := a.columns(c, 168)
	open := a.radar.open
	ui.Row(c.Key("radar-head")).AlignItems(ui.Center).Gap(10).Margin(34, 0, 14, 0).Children(func() {
		ui.Text(c, "私人雷达").FontSize(19).FontWeight(700).LetterSpacing(-0.3)
		ui.Text(c, "每天更新").FontSize(13).FontWeight(600).TextColor(p.faint).Margin(3, 0, 0, 0)
		ui.Box(c).Grow(1)
		if len(radars) <= cols {
			return
		}
		label := fmt.Sprintf("展开全部 %d", len(radars))
		if open {
			label = "收起"
		}
		b := ui.ButtonBase(c.Key("fold")).Height(28).PaddingX(12).Gap(6).Radius(999).AlignItems(ui.Center).FocusRing(false).
			Role(ui.RoleDisclosure).Label(label)
		b.Expanded(open)
		bg := p.hover
		if b.Hovered() {
			bg = p.press
		}
		b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
		b.Children(func() {
			ui.Text(c, label).FontSize(12.5).SingleLine()
			turn := float32(0)
			if open {
				turn = 180
			}
			ui.Icon(c, iconDown).Size(12, 12).TextColor(p.muted).Rotate(b.AnimateTo("turn", turn, motionFast))
		})
		if b.Clicked() {
			a.radar.open = !open
		}
	})
	shown := radars
	if !open {
		shown = radars[:min(cols, len(radars))]
	}
	ui.Grid(c.Key("radars")).Columns(cols).GapX(20).GapY(28).Children(func() {
		for _, r := range shown {
			pl := a.radar.lists[r.id]
			if a.coverCard(c.Key(r.id), r.name, pl.Cover(), 0, radarTitle(r, pl), func() { a.loadPlaylist(r.id, true) }) {
				a.open(route{page: pagePlaylist, id: r.id})
			}
		}
	})
}
