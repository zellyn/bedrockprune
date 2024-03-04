package lerp

import (
	"time"

	"github.com/chewxy/math32"
	"golang.org/x/exp/constraints"
)

type number interface {
	constraints.Integer | constraints.Float
}

type Lerp[T number] interface {
	At(float32) T
	Target() T
}

type TimeLerp[T number] interface {
	At(time.Time) T
	Done(time.Time) bool
	Target() T
}

type Curve func(float32) float32

type lerp[T number] struct {
	from  T
	to    T
	curve Curve
}

func (l lerp[T]) At(when float32) T {
	diff := l.to - l.from
	var add T
	if when <= 0 {
		return l.from
	} else if when > 1 {
		return l.to
	} else {
		amount := when
		if l.curve != nil {
			amount = l.curve(when)
		}
		add = T(float32(diff) * amount)
	}

	return l.from + add
}

func (l lerp[T]) Target() T {
	return l.to
}

type timeLerp[T number] struct {
	lerp     Lerp[T]
	start    time.Time
	end      time.Time
	duration time.Duration
}

func (tl timeLerp[T]) At(when time.Time) T {
	passed := float32(when.Sub(tl.start))
	return tl.lerp.At(passed / float32(tl.duration))
}

func (tl timeLerp[T]) Done(when time.Time) bool {
	return when.After(tl.end)
}

func (tl timeLerp[T]) Target() T {
	return tl.lerp.Target()
}

// NewLerp returns a new
func NewLerp[T number](from, to T, curve Curve) Lerp[T] {
	return lerp[T]{
		from:  from,
		to:    to,
		curve: curve,
	}
}

func NewTimeLerp[T number](from, to T, start time.Time, duration time.Duration, curve Curve) TimeLerp[T] {
	return timeLerp[T]{
		lerp:     NewLerp[T](from, to, curve),
		start:    start,
		end:      start.Add(duration),
		duration: duration,
	}
}

func Pow2(at float32) float32 {
	return math32.Pow(2, at) - 1
}

func Log2(at float32) float32 {
	return math32.Log2(at + 1)
}
