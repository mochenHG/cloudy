package app

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// The day's songs in the styles the listener picked, the official client's
// 风格推荐: the first card of the day's playlists on the home page, opening a page of the
// songs, whose styles a dialog changes.

// styleState is the day's style songs, and the dialog that picks their
// styles.
type styleState struct {
	songs   netease.StyleSongs
	loaded  bool
	loading bool
	err     string

	// The dialog: the categories of styles, the one shown, and the tags
	// picked, all of one category.
	open        bool
	cats        []netease.StyleCategory
	catsLoading bool
	catsErr     string
	tab         int64
	cat         int64
	picked      []int64
	saving      bool
	saveErr     string
}

var (
	dailyStyleSongs = netease.DailyStyleSongs
	styleCategories = netease.StyleCategories
	saveStyleTags   = netease.SaveStyleTags
)

// loadStyle loads the day's style songs for whoever is signed in.
func (a *app) loadStyle() {
	if a.profile == nil {
		a.style = styleState{}
		return
	}
	if a.style.loading {
		return
	}
	a.style.loading, a.style.err = true, ""
	gen := a.accountGen
	go func() {
		songs, err := dailyStyleSongs()
		a.post(func() {
			if gen != a.accountGen {
				return
			}
			s := &a.style
			s.loading = false
			if err != nil {
				s.err = "风格推荐加载失败"
				return
			}
			s.songs, s.loaded = songs, true
		})
	}()
}

// styleNames names the styles picked, as "R&B · 欧美流行".
func styleNames(picked netease.StyleCategory) string {
	names := make([]string, 0, len(picked.Tags))
	for _, t := range picked.Tags {
		names = append(names, t.Name)
	}
	return strings.Join(names, " · ")
}

// styleCover is the cover of the style songs: their first song's.
func styleCover(songs []netease.Track) string {
	for _, tr := range songs {
		if tr.Album.PicURL != "" {
			return tr.Album.PicURL
		}
	}
	return ""
}

// styleCardShown reports whether the home page has the style songs' card.
func (a *app) styleCardShown() bool {
	return a.profile != nil && a.style.loaded && len(a.style.songs.Songs) > 0
}

// recommendedGrid is the day's playlists picked for the listener, with the
// style songs' card first.
func (a *app) recommendedGrid(c *ui.Context) {
	cols := a.columns(c, 168)
	cell := a.gridCell(c, cols)
	style := a.styleCardShown()
	n := cols * 2
	if style {
		n -= 2 // its card is two wide
	}
	// 私人雷达 shows in the radars' section, under this one.
	recs := slices.DeleteFunc(slices.Clone(a.recPlaylists), func(pl netease.Playlist) bool { return isRadar(pl.ID) })
	recs = recs[:min(len(recs), n)]
	ui.Grid(c).Columns(cols).GapX(20).GapY(28).Children(func() {
		if style {
			a.styleCard(c.Key("style-songs"), cell)
		}
		for i := range recs {
			a.playlistCard(c.Key(recs[i].ID), &recs[i])
		}
	})
}

// gridCell is how wide a card is in a grid of cols on the page.
func (a *app) gridCell(c *ui.Context, cols int) float32 {
	w, _ := c.Size()
	return (w - a.sideWidth(w) - 2*pagePad - float32(cols-1)*20) / float32(cols)
}

// The style card's measures.
const (
	stylePad        = 18
	styleRadius     = 20
	styleSongRow    = 30
	styleHead       = 58 // its label and title, over its songs
	styleRoomy      = 260 // its songs' side this wide has room for how many
	styleArtists    = 210 // and this wide for their singers
)

// styleCard is the style songs' card, one card across two of the grid's
// cells, in the colors of their cover as the daily card is: the cover, the
// styles and as many of the songs as fit, each of which plays from there.
// The rest of it opens their page.
func (a *app) styleCard(c *ui.Context, cell float32) {
	s := &a.style
	songs := s.songs.Songs
	cover := styleCover(songs)
	art := a.art.get(netease.Sized(cover, 400), true)
	// As tall as a row of cards whose names take a line: a grid stretches
	// it to the row's tallest, never the row to it.
	tall := cell + 28
	inner := tall - 2*stylePad
	// A narrow card gives its songs some of the cover's room.
	room := 2*cell + 20 - 2*stylePad
	artSize := min(inner, room*0.42)
	side := room - artSize - 18
	card := ui.ButtonBase(c).ColumnSpan(2).MinHeight(tall).Radius(styleRadius).
		Padding(stylePad, stylePad).Gap(18).FocusRing(false).Label("风格推荐").TextColor(white)
	hovered := card.Hovered()
	still := c.Preferences().ReduceMotion
	deep, rich, glow := dailyColors(art)
	fade := ui.Animation{Duration: 600 * time.Millisecond}
	deep, rich, glow = card.AnimateColor("deep", deep, fade), card.AnimateColor("rich", rich, fade), card.AnimateColor("glow", glow, fade)
	card.Background(deep)
	// Under the pointer it rises a little and glows in its own colors, and
	// a sheen sweeps across it once, as on the daily card.
	lift := card.AnimateTo("lift", b2f(hovered), ui.Snappy)
	card.Shadow(0, 10+8*lift, 26+16*lift, -12, a.pal.cardShade.Mix(rich.Alpha(0.6), lift))
	pose := ui.Pose{}
	switch {
	case card.Pressed():
		pose.Scale = 0.99
	case hovered && !still:
		pose.Y = -3
	}
	card.AnimatePose(pose, ui.Snappy)
	sweep := float32(0)
	if hovered && !still {
		sweep = card.AnimateTo("sweep", 1, ui.Animation{Duration: 1100 * time.Millisecond, Ease: ui.EaseInOut})
	} else {
		card.Jump("sweep", 0)
	}
	blurred := backdrop(art)
	ours := a.queueIs(songs)
	cur := a.pl.Current()
	playing := a.pl.Playing()

	var artAt ui.Rect
	artBox := func(size float32) {
		box := ui.Box(c.Key("art")).Width(size).Height(size).Shrink(0).Radius(14).Shadow(0, 8, 22, -6, ui.RGBA(0, 0, 0, 0.4))
		artAt = box.Bounds()
		pose := ui.Pose{}
		if hovered {
			pose = ui.Pose{Scale: 1.02}
		}
		box.AnimatePose(pose, ui.Snappy)
		box.Children(func() {
			a.coverFill(c, cover, 400, 14)
			a.coverPlayButton(c, hovered, func() { a.playList(songs, false) })
		})
	}
	head := func() {
		ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(white.Alpha(0.72)).Children(func() {
					ui.Icon(c, iconWand).Size(13, 13)
					ui.Text(c, "风格推荐").FontSize(12).FontWeight(650).SingleLine().Shrink(0)
					if side >= styleRoomy {
						ui.Textf(c, "· 今日 %d 首", len(songs)).FontSize(12).SingleLine().Shrink(1).MinWidth(0)
					}
				})
				title := styleNames(s.songs.Picked)
				if title == "" {
					title = "按你选的风格挑选"
				}
				ui.Text(c, title).FontSize(20).FontWeight(800).LetterSpacing(-0.4).SingleLine()
			})
			change := ui.ButtonBase(c.Key("change")).Height(26).PaddingX(10).Gap(5).Radius(999).AlignItems(ui.Center).Shrink(0).
				FocusRing(false).Label("换风格")
			bg := white.Alpha(0.14)
			if change.Hovered() {
				bg = white.Alpha(0.24)
			}
			change.Background(change.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
			change.Children(func() {
				ui.Icon(c, iconWand).Size(12, 12)
				ui.Text(c, "换风格").FontSize(12).SingleLine()
			})
			if change.Clicked() {
				a.openStylePicker()
			}
		})
	}
	list := func(room, width float32) {
		shown := min(len(songs), max(1, int(room/styleSongRow)))
		ui.Column(c.Key("songs")).Children(func() {
			for i := range songs[:shown] {
				tr := &songs[i]
				current := ours && cur != nil && cur.ID == tr.ID
				label := "播放 " + tr.Name
				if current && playing {
					label = "暂停 " + tr.Name
				}
				row := ui.ButtonBase(c.Key(tr.ID)).Height(styleSongRow).Gap(8).PaddingX(6).Margin(0, -6).Radius(8).AlignItems(ui.Center).Justify(ui.Start).
					FocusRing(false).Label(label)
				bg := white.Alpha(0)
				if row.Hovered() {
					bg = white.Alpha(0.12)
				}
				row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
				row.Children(func() {
					a.pickMark(c, fmt.Sprint(i+1), white.Alpha(0.5), 22, row.Hovered(), current, playing)
					ui.Text(c, tr.Name).FontSize(13).FontWeight(600).SingleLine().Shrink(1).MinWidth(0)
					if width >= styleArtists {
						ui.Text(c, tr.ArtistNames()).FontSize(12).TextColor(white.Alpha(0.62)).SingleLine().Shrink(1).MinWidth(0).MaxWidth(width * 0.4)
					}
				})
				if row.Clicked() {
					if current {
						a.pl.TogglePlay()
					} else {
						if a.pl.Shuffle() {
							a.pl.ToggleShuffle()
						}
						a.pl.PlayQueue(songs, i)
					}
				}
			}
		})
	}

	card.Draw(func(p *ui.Painter, r ui.Rect) {
		// The glow is behind the cover.
		drawStyleBackdrop(p, r, blurred, deep, rich, glow, r.X+stylePad+artSize/2, r.Y+r.H/2, lift, sweep)
	})
	card.Children(func() {
		artBox(artSize)
		ui.Column(c.Key("side")).Grow(1).Basis(0).MinWidth(0).Height(inner).Gap(8).Children(func() {
			head()
			list(inner-styleHead-8, side)
		})
	})
	if card.Clicked() {
		a.flyFrom(c, cover, artAt, 14)
		a.open(route{page: pageStyle})
	}
}

// drawStyleBackdrop paints the style card in r, as the daily card is
// painted: a gradient of the cover's colors with the cover blurred faintly
// through it, deeper at the right, where its songs are, and a glow behind
// the cover, whose middle is at cx, cy, brighter and wider as lit goes to
// 1; and, as sweep goes from 0 to 1, a sheen crossing it.
func drawStyleBackdrop(p *ui.Painter, r ui.Rect, blurred *ui.Bitmap, deep, rich, glow ui.Color, cx, cy, lit, sweep float32) {
	const angle = 90
	p.Clip(r, styleRadius, func() {
		p.FillGradient(r, ui.LinearGradient{Angle: angle + 10, Oklab: true, Stops: []ui.GradientStop{
			{Color: rich, At: 0}, {Color: deep, At: 0.8},
		}}, 0)
		if blurred != nil {
			p.ImageRegion(blurred, ui.Rect{W: 48, H: 48}, coverRect(r, 48, 48), ui.ImageOptions{Tint: white.Alpha(0.24)})
		}
		p.FillGradient(r, ui.LinearGradient{Angle: angle, Stops: []ui.GradientStop{
			{Color: deep.Alpha(0), At: 0.25}, {Color: deep.Alpha(0.8), At: 1},
		}}, 0)
		g := 50 + 20*lit
		p.Shadow(ui.Rect{X: cx - g, Y: cy - g, W: 2 * g, H: 2 * g}, g, 0, 0, 120+40*lit, 30, glow.Alpha(0.22+0.18*lit))
		// Light from above, as the pointer comes.
		p.Shadow(ui.Rect{X: r.X + r.W*0.3, Y: r.Y - 80, W: r.W * 0.4, H: 40}, 20, 0, 0, 120, 0, white.Alpha(0.1*lit))
		if sweep > 0 && sweep < 1 {
			at := -0.25 + 1.5*sweep
			p.FillGradient(r, ui.LinearGradient{Angle: 110, Stops: []ui.GradientStop{
				{Color: white.Alpha(0), At: clamp01(at - 0.12)},
				{Color: white.Alpha(0.13 * (1 - sweep*0.5)), At: clamp01(at)},
				{Color: white.Alpha(0), At: clamp01(at + 0.12)},
			}}, 0)
		}
	})
	p.Stroke(r, white.Alpha(0.1+0.1*lit), styleRadius, 1)
}

// stylePage is the day's songs in the styles picked, with a button that
// changes those.
func (a *app) stylePage(c *ui.Context) {
	s := &a.style
	if a.profile == nil {
		a.centered(c, "登录后才能使用风格推荐", nil)
		return
	}
	loading := !s.loaded
	if loading && !s.loading {
		err := s.err
		if err == "" {
			err = "暂时没有风格推荐"
		}
		a.centered(c, err, a.loadStyle)
		return
	}
	songs := s.songs.Songs
	sh := a.pageShimmer(c)
	header := func(c *ui.Context) {
		p := a.pal
		ui.Row(c).Gap(28).AlignItems(ui.End).Padding(0, 0, 26, 0).Children(func() {
			a.headCover(c, styleCover(songs))
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, "每日推荐").FontSize(12).FontWeight(600).TextColor(p.accent)
				ui.Text(c, "风格推荐").FontSize(30).FontWeight(800).LetterSpacing(-0.6)
				if loading {
					sh.text(c, 13, 1.3).Width(240)
					return
				}
				ui.Text(c, fmt.Sprintf("%d 首 · 按你选的风格挑选，每天更新", len(songs))).FontSize(13).TextColor(p.muted)
				ui.Row(c).Gap(10).Margin(8, 0, 0, 0).Children(func() {
					if pillButton(c.Key("play"), iconPlay, "播放全部", p.accent, p.onAccent).Clicked() {
						a.playList(songs, false)
					}
					if pillButton(c.Key("shuffle"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).Clicked() {
						a.playList(songs, true)
					}
					label := styleNames(s.songs.Picked)
					if label == "" {
						label = "选择风格"
					}
					if pillButton(c.Key("styles"), iconWand, label, p.hover.Over(p.bg), p.text).Label("选择风格：" + label).Clicked() {
						a.openStylePicker()
					}
					a.batchEntry(c)
				})
			})
		})
	}
	if loading {
		a.trackSkeleton(c, sh, true, header)
		return
	}
	a.trackRows(c, songs, true, header)
}

// openStylePicker opens the dialog on the styles picked, loading what can
// be picked if it has not.
func (a *app) openStylePicker() {
	s := &a.style
	s.open, s.saveErr = true, ""
	s.cat = s.songs.Picked.ID
	s.picked = s.picked[:0]
	for _, t := range s.songs.Picked.Tags {
		s.picked = append(s.picked, t.ID)
	}
	s.tab = s.cat
	if len(s.cats) > 0 || s.catsLoading {
		a.styleTabFallback()
		return
	}
	s.catsLoading, s.catsErr = true, ""
	gen := a.accountGen
	go func() {
		cats, err := styleCategories()
		a.post(func() {
			if gen != a.accountGen {
				return
			}
			s.catsLoading = false
			if err != nil || len(cats) == 0 {
				s.catsErr = "风格列表加载失败"
				return
			}
			s.cats = cats
			a.styleTabFallback()
		})
	}()
}

// styleTabFallback shows the first category if the one picked in isn't
// one there is.
func (a *app) styleTabFallback() {
	s := &a.style
	if len(s.cats) > 0 && !slices.ContainsFunc(s.cats, func(cat netease.StyleCategory) bool { return cat.ID == s.tab }) {
		s.tab = s.cats[0].ID
	}
}

// pickStyle picks tag of category cat, or unpicks it. Tags are picked in
// one category: one of another starts the picking again.
func (a *app) pickStyle(cat, tag int64) {
	s := &a.style
	if s.cat != cat {
		s.cat, s.picked = cat, []int64{tag}
		return
	}
	if i := slices.Index(s.picked, tag); i >= 0 {
		s.picked = slices.Delete(s.picked, i, i+1)
		return
	}
	s.picked = append(s.picked, tag)
}

// saveStyles keeps the styles picked with the account, and loads the songs
// in them.
func (a *app) saveStyles() {
	s := &a.style
	if s.saving || len(s.picked) == 0 {
		return
	}
	s.saving, s.saveErr = true, ""
	cat, tags := s.cat, slices.Clone(s.picked)
	gen := a.accountGen
	go func() {
		err := saveStyleTags(cat, tags)
		var songs netease.StyleSongs
		if err == nil {
			songs, err = dailyStyleSongs()
		}
		a.post(func() {
			if gen != a.accountGen {
				return
			}
			s.saving = false
			if err != nil {
				s.saveErr = "保存失败，请稍后再试"
				return
			}
			s.songs, s.loaded, s.open = songs, true, false
		})
	}()
}

// stylePickerDialog picks the styles of the style songs: a tab for each
// category, its tags under it.
func (a *app) stylePickerDialog(c *ui.Context) {
	s := &a.style
	p := a.pal
	a.dialog(c, "style-picker", 0.35, &s.open, func(panel ui.Element) {
		a.pane(panel, 26, false)
		ink := a.inkOn(false)
		panel.Width(560).TextColor(ink.text)
		dialogHeader(c, ink, "选择风格", "同一类里可以多选，和网易云官方客户端同步", func() { s.open = false })
		ui.Column(c).Padding(0, 28).Gap(16).MinHeight(250).Children(func() {
			switch {
			case s.catsLoading:
				ui.Text(c, "正在加载…").FontSize(13).TextColor(ink.muted)
				return
			case s.catsErr != "":
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, s.catsErr).FontSize(13).TextColor(ink.muted)
					if settingButton(c.Key("retry"), ink, "重试", nil, false, false).Clicked() {
						a.openStylePicker()
					}
				})
				return
			}
			ui.Row(c.Key("tabs")).Gap(6).Children(func() {
				for _, cat := range s.cats {
					on := cat.ID == s.tab
					b := ui.ButtonBase(c.Key(cat.ID)).Height(32).PaddingX(14).Radius(999).Center().
						FocusRing(false).Label("风格分类：" + cat.Name).Selected(on)
					bg, fg := ui.Transparent, ink.muted
					switch {
					case on:
						bg, fg = ink.hover, ink.text
					case b.Hovered():
						fg = ink.text
					}
					b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
					b.Children(func() {
						label := cat.Name
						if n := a.pickedIn(cat.ID); n > 0 {
							label = fmt.Sprintf("%s %d", cat.Name, n)
						}
						ui.Text(c, label).FontSize(14).FontWeight(650).TextColor(fg).SingleLine()
					})
					if b.Clicked() {
						s.tab = cat.ID
					}
				}
			})
			for _, cat := range s.cats {
				if cat.ID != s.tab {
					continue
				}
				ui.Grid(c.Key(fmt.Sprint("tags-", cat.ID))).Columns(5).Gap(10).Children(func() {
					for _, t := range cat.Tags {
						on := s.cat == cat.ID && slices.Contains(s.picked, t.ID)
						b := ui.ButtonBase(c.Key(t.ID)).Height(34).Radius(999).Center().FocusRing(false).
							Label("风格：" + t.Name).Selected(on)
						bg, fg := ink.hover, ink.text
						switch {
						case on:
							bg, fg = p.accent, p.onAccent
						case b.Hovered():
							bg = ink.hover.Mix(ink.text, 0.08)
						}
						b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
						b.Children(func() {
							ui.Text(c, t.Name).FontSize(13).FontWeight(600).TextColor(fg).SingleLine()
						})
						if b.Clicked() {
							a.pickStyle(cat.ID, t.ID)
						}
					}
				})
			}
			if s.saveErr != "" {
				ui.Text(c, s.saveErr).FontSize(12.5).TextColor(dangerColor)
			}
		})
		dialogFooter(c, func() {
			if settingButton(c.Key("cancel"), ink, "取消", nil, false, false).Clicked() {
				s.open = false
			}
			label := "确认"
			if s.saving {
				label = "正在保存…"
			}
			if primaryButton(c.Key("ok"), label, nil, p, len(s.picked) == 0 || s.saving || len(s.cats) == 0).Clicked() {
				a.saveStyles()
			}
		})
	})
}

// pickedIn says how many tags of category cat are picked.
func (a *app) pickedIn(cat int64) int {
	if a.style.cat != cat {
		return 0
	}
	return len(a.style.picked)
}
