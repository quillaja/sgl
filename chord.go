package sgl

import (
	"sort"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Chord is an input "gesture", which may be one or more keys (eg CTRL+ALT+T).
type Chord struct {
	lastPressed time.Time
	Keys        []glfw.Key // List of keys to be down to execute this chord
	Execute     func()     // function to execute
	Wait        float64    // Wait time (seconds) between sucessive allowable executions
	Stop        bool       // When set, no further chords will be executed after this one has been
	// Mouse       []glfw.MouseButton
}

// Match determines whether or not the keys for this chord are pressed and if
// the chord's Wait time has elapsed.
func (c *Chord) Match(win *glfw.Window) bool {
	// check wait time
	if time.Since(c.lastPressed).Seconds() < c.Wait {
		return false
	}

	for i := range c.Keys {
		if win.GetKey(c.Keys[i]) != glfw.Press {
			return false
		}
	}
	// for i := range c.Mouse {
	// 	if win.GetMouseButton(c.Mouse[i]) != glfw.Press {
	// 		return false
	// 	}
	// }

	c.lastPressed = time.Now() // reset
	return true
}

// ChordSet is a logic grouping of (related) Chords.
type ChordSet []Chord

// Match returns first Chord in the set that matches the current
// key state.
func (cs ChordSet) Match(win *glfw.Window) *Chord {
	var i int
	for i = 0; i < len(cs); i++ {
		if cs[i].Match(win) {
			return &cs[i]
		}
	}
	return nil
}

// Execute runs the function for each Chord that matches
// the current key state. Execution of chords will stop when
// the first chord is encountered with its "Stop" member set to true.
func (cs ChordSet) Execute(win *glfw.Window) {
	var done bool
	for i := 0; i < len(cs) && !done; i++ {
		if cs[i].Match(win) {
			cs[i].Execute()
			done = cs[i].Stop
		}
	}
}

// Sort called sort.Sort() on the ChordSet, returning the same
// ChordSet for convenience.
func (cs ChordSet) Sort() ChordSet {
	sort.Sort(cs)
	return cs
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

// CombineSets makes a slice of ChordSets for convenience, sorting each one.
func CombineSets(sets ...ChordSet) []ChordSet {
	for i := range sets {
		sort.Sort(sets[i])
	}
	return sets
}

// ExecuteSets calls Execute() on each ChordSet.
func ExecuteSets(sets []ChordSet, win *glfw.Window) {
	// using defer here so all the "searching" can be done first,
	// then all the actual exections
	for i := range sets {
		sets[i].Execute(win)
	}
}
