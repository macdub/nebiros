package Utils

import (
	"slices"
	"strings"
)

const (
	H  string = "─"
	V  string = "│"
	VR string = "├"
	VL string = "┤"
	HD string = "┬"
	HU string = "┴"
	X  string = "┼"
	TL string = "╭"
	TR string = "╮"
	BR string = "╯"
	BL string = "╰"
)

type CommandType int

const (
	COMMAND CommandType = iota
	KUBERNETES
)

func StringSliceContains(haystack []string, needle string) bool {
	inSlice := slices.ContainsFunc(haystack, func(c string) bool {
		return strings.Contains(c, needle)
	})
	return inSlice
}
