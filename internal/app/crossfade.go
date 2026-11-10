package app

import "time"

// crossfadeChoices are how many seconds songs may fade into each other.
var crossfadeChoices = []int{1, 2, 3, 5, 8}

// crossfadeSec is how many seconds songs fade into each other, as set.
func (a *app) crossfadeSec() int {
	if n := a.settings.CrossfadeSec; n > 0 {
		return n
	}
	return 3
}

// crossfade is how long songs fade into each other, 0 with the setting
// off.
func (a *app) crossfade() time.Duration {
	if !a.settings.Crossfade {
		return 0
	}
	return time.Duration(a.crossfadeSec()) * time.Second
}
