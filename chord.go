package sgl

import (
	"sort"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type Chord struct {
	Keys    []glfw.Key
	Execute func()
	Wait    float64
	t       float64
}

func (c *Chord) Match(win *glfw.Window, dt float64) bool {
	// decrease time
	if c.t > 0 {
		c.t -= dt
		return false
	}
	c.t = 0 // prevent going negative

	for i := range c.Keys {
		if win.GetKey(c.Keys[i]) != glfw.Press {
			return false
		}
	}

	c.t = c.Wait // reset
	return true
}

type ChordSet []Chord

func (cs ChordSet) Match(win *glfw.Window, dt float64) *Chord {
	var i int
	for i = 0; i < len(cs); i++ {
		if cs[i].Match(win, dt) {
			return &cs[i]
		}
	}
	return nil
}

func (cs ChordSet) Execute(win *glfw.Window, dt float64) {
	if match := cs.Match(win, dt); match != nil {
		match.Execute()
	}
}

// Len is the number of elements in the collection.
func (cs ChordSet) Len() int {
	return len(cs)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (cs ChordSet) Less(i int, j int) bool {
	dLen := len(cs[i].Keys) - len(cs[j].Keys)
	if dLen == 0 {
		// if same length, order by integer value of keys
		for k := range cs[i].Keys {
			if cs[i].Keys[k] < cs[j].Keys[k] {
				return true
			}
		}
	}
	return dLen > 0
}

// Swap swaps the elements with indexes i and j.
func (cs ChordSet) Swap(i int, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

func CombineSets(sets ...ChordSet) []ChordSet {
	for i := range sets {
		sort.Sort(sets[i])
	}
	return sets
}

func ExecuteSets(sets []ChordSet, win *glfw.Window, dt float64) {
	// using defer here so all the "searching" can be done first,
	// then all the actual exections
	for i := range sets {
		if match := sets[i].Match(win, dt); match != nil {
			defer match.Execute()
		}
	}
}
