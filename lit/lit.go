package lit

import (
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

// BreakIter is a special error value that can be returned from iterators.
// It indicates that the iteration should be stopped even though no actual failure occurred.
var BreakIter = cor.StrError("break iter")

// Deopt returns the wrapped literal if l is an optional literal, otherwise it returns l as-is.
func Deopt(l Lit) Lit {
	if o, ok := l.(Opter); ok {
		return o.Some()
	}
	return l
}

// Lit is the common interface for all literal adapters.
// A nil Lit represents an absent value.
type Lit interface {
	// Typ returns the defined type of the literal.
	Typ() typ.Type
	// IsZero returns whether the literal value is the zero value.
	IsZero() bool
	// WriteBfr writes to a bfr ctx either as strict JSON or xelf representation.
	WriteBfr(*bfr.Ctx) error
	// String returns the xelf representation as string.
	String() string
	// MarshalJSON returns the JSON representation as bytes.
	MarshalJSON() ([]byte, error)
}

// Opter is the interface for literals with an optional type.
type Opter interface {
	Lit
	// Some returns the wrapped literal or nil
	Some() Lit
}

// Proxr is the encapsulates the extra method-set of proxy literals.
// It is used only for easier interface composition before Go 1.13.
type Proxr interface {
	// New returns a zero instance of the proxy literal.
	New() Proxy
	// Ptr returns a pointer to the underlying go value as interface.
	Ptr() interface{}
	// Assign assigns the value of the given literal or returns an error.
	// The literal must be valid literal of the same type.
	Assign(Lit) error
}

// Idxr is encapsulates the extra method-set of indexer literals.
// It is used only for easier interface composition before Go 1.13.
type Idxr interface {
	// Idx returns the literal of the element at idx or an error.
	Idx(idx int) (Lit, error)
	// SetIdx sets the element value at idx to l and returns the indexer or an error.
	SetIdx(idx int, l Lit) (Indexer, error)
	// IterIdx iterates over elements, calling iter with the elements index and literal value.
	// If iter returns an error the iteration is aborted.
	IterIdx(iter func(int, Lit) error) error
}

// Keyr is encapsulates the extra method-set of keyer literals.
// It is only used for easier interface composition before Go 1.13.
type Keyr interface {
	// Keys returns a string slice of all keys.
	Keys() []string
	// Key returns the literal of the element with key key or an error.
	Key(key string) (Lit, error)
	// SetKey sets the elements value with key to l and returns the keyer or an error.
	SetKey(key string, l Lit) (Keyer, error)
	// IterKey iterates over elements, calling iter with the elements key and literal value.
	// If iter returns an error the iteration is aborted.
	IterKey(iter func(string, Lit) error) error
}

// Indexer is the common interface for container literals with elements accessible by index.
type Indexer interface {
	Lit
	Idxr
	// Len returns the number of contained elements.
	Len() int
}

// Keyer is the common interface for container literals with elements accessible by key.
type Keyer interface {
	Lit
	Keyr
	// Len returns the number of contained elements.
	Len() int
}

// Proxy is the common interface for proxies and some adapter pointers that can be assigned to.
type Proxy interface {
	Lit
	Proxr
}

// Numeric is the common interface for numeric literals.
type Numeric interface {
	Lit
	// Num returns the numeric value of the literal as float64.
	Num() float64
	// Val returns the simple go value representing this literal.
	// The type is either bool, int64, float64, time.Time or time.Duration
	Val() interface{}
}

// Character is the common interface for character literals.
type Character interface {
	Lit
	// Char returns the character format of the literal as string.
	Char() string
	// Val returns the simple go value representing this literal.
	// The type is either string, []byte, [16]byte, time.Time or time.Duration.
	Val() interface{}
}

// Appender is the common interface for list literals.
type Appender interface {
	Indexer
	// Append appends the given literals and returns a new appender or an error
	Append(...Lit) (Appender, error)
	// Element returns a newly created proxy of the element type or an error.
	Element() (Proxy, error)
}

// Dictionary is the interface for dict literals.
type Dictionary interface {
	Keyer
	// Element returns a newly created proxy of the element type or an error.
	Element() (Proxy, error)
}

// Record is the interface for record literals.
type Record interface {
	Lit
	Idxr
	Keyr
	// Len returns the number of fields.
	Len() int
}

// MarkSpan is a marker interface. When implemented on an int64 indicates a span type.
type MarkSpan interface{ Seconds() float64 }

// MarkBits is a marker interface. When implemented on an unsigned integer indicates a bits type.
type MarkBits interface{ Bits() map[string]int64 }

// MarkEnum is a marker interface. When implemented on a string or integer indicates an enum type.
type MarkEnum interface{ Enums() map[string]int64 }
