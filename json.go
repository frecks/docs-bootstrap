package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// object is a JSON object that keeps its keys in order, so that merging into
// a settings file doesn't reorder it.
type object struct {
	keys   []string
	values map[string]any
}

func newObject() *object {
	return &object{values: map[string]any{}}
}

func (o *object) set(key string, value any) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *object) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, key := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		if err := encodeCompact(&b, key); err != nil {
			return nil, err
		}
		b.WriteByte(':')
		if err := encodeCompact(&b, o.values[key]); err != nil {
			return nil, err
		}
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func encodeCompact(b *bytes.Buffer, v any) error {
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	// Encode ends with a newline.
	b.Truncate(b.Len() - 1)
	return nil
}

// decodeJSON decodes text into *object, []any, string, json.Number, bool, and nil values.
func decodeJSON(text string) (any, error) {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	value, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("extra data after the JSON value")
	}
	return value, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		obj := newObject()
		for dec.More() {
			key, err := dec.Token()
			if err != nil {
				return nil, err
			}
			value, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			obj.set(key.(string), value)
		}
		_, err := dec.Token()
		return obj, err
	case json.Delim('['):
		list := []any{}
		for dec.More() {
			value, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			list = append(list, value)
		}
		_, err := dec.Token()
		return list, err
	}
	return token, nil
}

// get returns an object's value for key, or nil if v isn't an object.
func get(v any, key string) any {
	if obj, ok := v.(*object); ok {
		return obj.values[key]
	}
	return nil
}

// index returns a list's item i, or nil if v isn't a list that long.
func index(v any, i int) any {
	if list, ok := v.([]any); ok && i < len(list) {
		return list[i]
	}
	return nil
}
