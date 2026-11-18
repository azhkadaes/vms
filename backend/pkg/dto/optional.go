package dto

import "encoding/json"

// Optional is a tri-state request field: omitted (Set false), explicit null (Null) or a
// value. It is for requests only (decision 15 room rule, note clearing); responses omit
// absent fields instead of sending null.
type Optional[T any] struct {
	Set   bool
	Null  bool
	Value T
}

// UnmarshalJSON runs only when the member is present, which is what marks it Set.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	var value T
	if string(data) == "null" {
		*o = Optional[T]{Set: true, Null: true, Value: value}
		return nil
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*o = Optional[T]{Set: true, Value: value}
	return nil
}
