package jsonpointerpos

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-openapi/jsonpointer"
	"github.com/stretchr/testify/require"
)

func TestBuildTokenTree(t *testing.T) {
	cases := []struct {
		name   string
		input  []string
		expect tokenTree
	}{
		{
			name:   "nil",
			input:  nil,
			expect: tokenTree{},
		},
		{
			name:   "empty",
			input:  []string{},
			expect: tokenTree{},
		},
		{
			name:  "many pointers",
			input: []string{"/foo", "/foo/a", "/foo/b", "/bar/a/b", "/"},
			expect: tokenTree{
				tk: "",
				children: map[string]*tokenTree{
					"foo": {
						tk: "foo",
						children: map[string]*tokenTree{
							"a": {
								tk: "a",
							},
							"b": {
								tk: "b",
							},
						},
					},
					"bar": {
						tk: "bar",
						children: map[string]*tokenTree{
							"a": {
								tk: "a",
								children: map[string]*tokenTree{
									"b": {
										tk: "b",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var ptrs []jsonpointer.Pointer
			for _, v := range tt.input {
				ptr, err := jsonpointer.New(v)
				require.NoError(t, err)
				ptrs = append(ptrs, ptr)
			}
			require.Equal(t, tt.expect, buildTokenTree(ptrs))
		})
	}
}

func TestOffsetValue(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		ptrs   []string
		length int
		expect tokenTree
	}{
		{
			name:   "empty object",
			input:  "{}",
			length: 2,
			expect: tokenTree{},
		},
		{
			name:   "empty array",
			input:  "[]",
			length: 2,
			expect: tokenTree{},
		},
		{
			name:   "empty object with non-exist ptr",
			input:  "{}",
			ptrs:   []string{"/foo"},
			length: 2,
			expect: tokenTree{
				children: map[string]*tokenTree{
					"foo": {
						tk: "foo",
					},
				},
			},
		},
		{
			name:   "simple object",
			input:  `{  "string" : "foo" , "number" : 123 , "float" : 1.23, "null" : null , "true" : true, "false" : false,  "obj" : {"x": 3}}`,
			ptrs:   []string{"/string", "/number", "/float", "/null", "/true", "/false", "/obj/x"},
			length: 121,
			expect: tokenTree{
				children: map[string]*tokenTree{
					"string": {
						tk:     "string",
						offset: ptr(14),
					},
					"number": {
						tk:     "number",
						offset: ptr(33),
					},
					"float": {
						tk:     "float",
						offset: ptr(49),
					},
					"null": {
						tk:     "null",
						offset: ptr(64),
					},
					"true": {
						tk:     "true",
						offset: ptr(80),
					},
					"false": {
						tk:     "false",
						offset: ptr(96),
					},
					"obj": {
						tk:     "obj",
						offset: ptr(112),
						children: map[string]*tokenTree{
							"x": {
								tk:     "x",
								offset: ptr(118),
							},
						},
					},
				},
			},
		},
		{
			name:   "simple array",
			input:  `[[1,2], [3,4]]`,
			ptrs:   []string{"/0/1"},
			length: 14,
			expect: tokenTree{
				children: map[string]*tokenTree{
					"0": {
						tk:     "0",
						offset: ptr(1),
						children: map[string]*tokenTree{
							"1": {
								tk:     "1",
								offset: ptr(4),
							},
						},
					},
				},
			},
		},
		{
			name:   "mix array index and object key",
			input:  `[[1, {"foo": ["a", "b"]}], [3, 4]]`,
			ptrs:   []string{"/0/1/foo/0"},
			length: 34,
			expect: tokenTree{
				children: map[string]*tokenTree{
					"0": {
						tk:     "0",
						offset: ptr(1),
						children: map[string]*tokenTree{
							"1": {
								tk:     "1",
								offset: ptr(5),
								children: map[string]*tokenTree{
									"foo": {
										tk:     "foo",
										offset: ptr(13),
										children: map[string]*tokenTree{
											"0": {
												tk:     "0",
												offset: ptr(14),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dec := json.NewDecoder(strings.NewReader(tt.input))
			dec.UseNumber()
			var ptrs []jsonpointer.Pointer
			for _, v := range tt.ptrs {
				ptr, err := jsonpointer.New(v)
				require.NoError(t, err)
				ptrs = append(ptrs, ptr)
			}
			tree := buildTokenTree(ptrs)
			length, err := offsetValue(dec, &tree)
			require.NoError(t, err)
			require.Equal(t, tt.length, length)
			require.Equal(t, tt.expect, tree)
		})
	}
}

func TestGetPositions(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		ptrs   []string
		expect map[string]JSONPointerPosition
	}{
		{
			name:   "empty object",
			input:  "{}",
			expect: nil,
		},
		{
			name:   "empty array",
			input:  "[]",
			expect: nil,
		},
		{
			name:   "empty object with non-exist ptr",
			input:  "{}",
			ptrs:   []string{"/foo"},
			expect: map[string]JSONPointerPosition{},
		},
		{
			name: "simple object",
			input: `
{
  "a": 1,
  "b": 2,
  "c": {
    "x": 3
  }
}`,
			ptrs: []string{"/b", "/c/x", "/non-exist"},
			expect: map[string]JSONPointerPosition{
				"/b": {
					Ptr: *newJSONPtr([]string{"b"}),
					Position: Position{
						Line:   4,
						Column: 8,
					},
				},
				"/c/x": {
					Ptr: *newJSONPtr([]string{"c", "x"}),
					Position: Position{
						Line:   6,
						Column: 10,
					},
				},
			},
		},
		{
			name: "simple array",
			input: `
[
  [1, 2],
  [3, 4]
]`,
			ptrs: []string{"/0/1"},
			expect: map[string]JSONPointerPosition{
				"/0/1": {
					Ptr: *newJSONPtr([]string{"0", "1"}),
					Position: Position{
						Line:   3,
						Column: 7,
					},
				},
			},
		},
		{
			name: "mix array index and object key",
			input: `
[
  [
    1,
    {
      "foo": ["a", "b"]
    }
  ],
  [3, 4]
]`,
			ptrs: []string{"/0/1/foo/0"},
			expect: map[string]JSONPointerPosition{
				"/0/1/foo/0": {
					Ptr: *newJSONPtr([]string{"0", "1", "foo", "0"}),
					Position: Position{
						Line:   6,
						Column: 15,
					},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var ptrs []jsonpointer.Pointer
			for _, v := range tt.ptrs {
				ptr, err := jsonpointer.New(v)
				require.NoError(t, err)
				ptrs = append(ptrs, ptr)
			}
			out, err := GetPositions(tt.input, ptrs)
			require.NoError(t, err)
			require.Equal(t, tt.expect, out)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
