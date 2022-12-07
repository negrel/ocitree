package components

// Named define object with a name.
type Named interface {
	Name() string
}

// Tagged is an object which has a tag.
type Tagged interface {
	Tag() string
}

// Identifier is an object which has an ID.
type Identifier interface {
	ID() string
}
