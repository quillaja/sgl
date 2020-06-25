package sgl

import (
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
	DeltaT      float64
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

// Cycler lets one easily cycle through a list of "whatever".
type Cycler struct {
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
