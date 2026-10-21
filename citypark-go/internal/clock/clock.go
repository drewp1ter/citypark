// Package clock provides the current time behind an interface so that feature
// code never calls time.Now directly and tests can pin the time.
package clock

import "time"

// Clock reports the current time.
type Clock interface{ Now() time.Time }

type system struct{ loc *time.Location }

func (s system) Now() time.Time { return time.Now().In(s.loc) }

// System returns the real time converted to loc.
func System(loc *time.Location) Clock { return system{loc: loc} }

type fixed struct{ t time.Time }

func (f fixed) Now() time.Time { return f.t }

// Fixed always returns t. Used in tests.
func Fixed(t time.Time) Clock { return fixed{t: t} }
