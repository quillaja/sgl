package sgl

import (
	"fmt"
	"sort"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

// linear interpolate a value between from and to at point t.
func lerp(t, from, to float32) float32 { return from + t*(to-from) }

// find a "t" value for x in the range min to max.
func invLerp(x, min, max float32) float32 {
	return (x - min) / (max - min)
}

// Timer keeps time and other similar info useful for an opengl render loop.
type Timer struct {
	TotalFrames uint64
	TotalTime   float64
	DeltaT      float64 // Seconds
	Start       time.Time
	Now         time.Time
}

// Reset the timer to an initial state. Should call once before the render loop.
func (t *Timer) Reset() {
	t.TotalFrames = 0
	t.DeltaT = 0
	t.Now = time.Now()
	t.Start = t.Now
}

// Update the timer with the current time. Call once each render loop.
func (t *Timer) Update() {
	t.TotalFrames++
	current := time.Now()
	t.DeltaT = current.Sub(t.Now).Seconds()
	t.Now = current
	t.TotalTime += t.DeltaT
}

// AvgFps gets the average framerate over the total program runtime (or
// since Reset() was called).
func (t *Timer) AvgFps() float64 {
	return float64(t.TotalFrames) / t.TotalTime
}

// Fps gets the instantaneous framerate of this render loop.
func (t *Timer) Fps() float64 {
	return 1.0 / t.DeltaT
}

// IsNthFrame returns true if the current frame number is on the "nth" since
// the timer was last reset. Just frame count mod n == 0.
// Example:
//  if timer.IsNthFrame(2) {
//  	// do something every other frame
//  }
func (t *Timer) IsNthFrame(n uint64) bool {
	return t.TotalFrames%n == 0
}

// Cycler lets one easily cycle through a list of "whatever". It's a
// simpler version of Selecter that doesn't name items and allows only
// relative (Next() and Previous()) selection changes.
type Cycler struct {
	Title   string
	Current int
	Things  []interface{}
}

// NewCycler creates a Cycler from items.
func NewCycler(items ...interface{}) *Cycler {
	return &Cycler{
		Things: items,
	}
}

// Get the current item.
func (c *Cycler) Get() interface{} { return c.Things[c.Current] }

// Next moves to the next item, or wraps around if at the end.
func (c *Cycler) Next() {
	c.Current++
	if c.Current == len(c.Things) {
		c.Current = 0
	}
}

// Previous moves to the previous item, or wraps around if at the begining.
func (c *Cycler) Previous() {
	c.Current--
	if c.Current == -1 {
		c.Current = len(c.Things) - 1
	}
}

// NamedItems is a slice of structs that pairs a Name string with any Item.
type NamedItems []selecteritem

// MakeItems creates a NamedItems from the provided "items". Items
// implementing fmt.Stringer use that for Name. Most other types use their
// (possibly truncated) go representation, which will be annotated with its
// type for types such as slices, structs, etc.
func MakeItems(items ...interface{}) NamedItems {
	list := make(NamedItems, 0, len(items))
	for i, item := range items {
		var name string
		switch v := item.(type) {
		case string:
			name = v
		case int, int8, uint8, int32, uint32, int64, uint64, float32, float64:
			name = fmt.Sprintf("%v", v)
		case fmt.Stringer:
			name = v.String()
		default:
			//name = fmt.Sprintf("%T %d", item, i)
			const maxlen = 42
			name = fmt.Sprintf("(%d) %T %+v", i, item, item)
			if len(name) > maxlen {
				name = name[:maxlen] + "..."
			}
		}
		list = append(list, selecteritem{
			Name: name,
			Item: item,
		})
	}
	return list
}

// Sort the NamedItems by Name, descending (alphabetical).
// Returns the NamedItems for "inline" use.
func (items NamedItems) Sort() NamedItems {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

type selecteritem struct {
	Name string
	Item interface{}
}

// TODO: make Selecter able to handle multiple selected items. Perhaps another
// slice of bool as "selected toggles"?

// Selecter lets someone create an indexed list of "whatever" with an associated name.
// The current selection can be changed absolutely (with Set()) or relatively (with Next()
// and Previous()). Only 1 item can be selected.
// This is helpful for use with imgui's combo box or list box.
type Selecter struct {
	Title   string
	Current int
	Things  []interface{}
	Names   []string
}

// NewSelecter creates a Selecter using the items to populate its Things and
// Names slices.
func NewSelecter(items NamedItems) *Selecter {
	s := Selecter{
		Things: make([]interface{}, len(items)),
		Names:  make([]string, len(items)),
	}
	for i := range items {
		s.Things[i] = items[i].Item
		s.Names[i] = items[i].Name
	}
	return &s
}

// Get the current item.
// example:
//  fmt.Println(selecter.Get().Name)
//	thing := selecter.Get().Item.(mytype)
func (s *Selecter) Get() selecteritem {
	return selecteritem{Item: s.Things[s.Current], Name: s.Names[s.Current]}
}

// Selected returns true if index is the current selection.
func (s *Selecter) Selected(index int) bool {
	return s.Current == index
}

// SelectedName performs a linear search on the Names slice for name
// and, if a match is found, returns true if it is the current selection.
func (s *Selecter) SelectedName(name string) bool {
	for i := range s.Names {
		if s.Names[i] == name {
			return s.Selected(i)
		}
	}
	return false
}

// Set the current selection. index is clamped to the
// bounds of the Things slice.
func (s *Selecter) Set(index int) {
	if index >= len(s.Things) {
		index = len(s.Things) - 1
	}
	if index < 0 {
		index = 0
	}
	s.Current = index
}

// SetName performs a linear search on the Names slice for name
// and, if a match is found, sets it to the current selection.
// No change is performed if a match is not found.
func (s *Selecter) SetName(name string) {
	for i := range s.Names {
		if s.Names[i] == name {
			s.Set(i)
			return
		}
	}
}

// Next changes the selection to the next item, or wraps around if at the end.
func (s *Selecter) Next() {
	s.Current++
	if s.Current == len(s.Things) {
		s.Current = 0
	}
}

// Previous changes the selection to the previous item, or wraps around if at the end.
func (s *Selecter) Previous() {
	s.Current--
	if s.Current == -1 {
		s.Current = len(s.Things) - 1
	}
}

// Animation is a function that takes a time delta and returns whether or
// not the animation has completed.
type Animation func(float32) bool

// AnimationMap holds animations keyed by some name.
type AnimationMap map[string]Animation

// Update every animation in the map, deleting those that are completed.
func (am AnimationMap) Update(dt float32) {
	for name, ani := range am {
		done := ani(dt)
		if done {
			delete(am, name)
		}
	}
}

// Has checks the animation map for an animation of the given name.
func (am AnimationMap) Has(name string) bool {
	_, has := am[name]
	return has
}

// Float32 inserts a new animation with "name" which animates the value from "from" to "to" over
// "durationSec" seconds.
func (am AnimationMap) Float32(name string, value *float32, durationSec, from, to float32) {
	var elapsed float32
	am[name] = func(dt float32) (done bool) {
		elapsed += dt
		t := mgl32.Clamp(elapsed/durationSec, 0, 1)
		*value = lerp(t, from, to)
		if elapsed > durationSec {
			return true
		}
		return false
	}
}

// Vec3f inserts a new animation with "name" which animates the value from "from" to "to" over
// "durationSec" seconds.
func (am AnimationMap) Vec3f(name string, value *mgl32.Vec3, durationSec float32, from, to mgl32.Vec3) {
	var elapsed float32
	am[name] = func(dt float32) (done bool) {
		elapsed += dt
		t := mgl32.Clamp(elapsed/durationSec, 0, 1)
		(*value)[0] = lerp(t, from[0], to[0])
		(*value)[1] = lerp(t, from[1], to[1])
		(*value)[2] = lerp(t, from[2], to[2])
		if elapsed > durationSec {
			return true
		}
		return false
	}
}
