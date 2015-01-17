package main

import (
	"sort"
	"strconv"
)

type Entry struct {
	Host  string            `json:"host"`
	Attrs map[string]string `json:"attrs"`
}

func Sort(field string, entries []*Entry) {
	switch field {
	case "uri":
		By(uriSort).Sort(entries)
	case "time":
		By(timeSort).Sort(entries)
	case "ip":
		By(addrSort).Sort(entries)
	default:
		By(hostSort).Sort(entries)
	}
}

type EntrySorter struct {
	entries []*Entry
	by      func(p1, p2 *Entry) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *EntrySorter) Len() int {
	return len(s.entries)
}

// Swap is part of sort.Interface.
func (s *EntrySorter) Swap(i, j int) {
	s.entries[i], s.entries[j] = s.entries[j], s.entries[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *EntrySorter) Less(i, j int) bool {
	return s.by(s.entries[i], s.entries[j])
}

// By is the type of a "less" function that defines the ordering of its Entry arguments.
type By func(p1, p2 *Entry) bool

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By) Sort(entries []*Entry) {
	ps := &EntrySorter{
		entries: entries,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

// Closures that order the entries
var hostSort = func(e1, e2 *Entry) bool {
	return e1.Host < e2.Host
}
var uriSort = func(e1, e2 *Entry) bool {
	return e1.Attrs["uri"] < e2.Attrs["uri"]
}

// this sorts the ip address as a string
var addrSort = func(e1, e2 *Entry) bool {
	return e1.Attrs["remoteAddr"] < e2.Attrs["remoteAddr"]
}
var timeSort = func(e1, e2 *Entry) bool {
	val1, err1 := strconv.Atoi(e1.Attrs["requestProcessingTime"])
	val2, err2 := strconv.Atoi(e2.Attrs["requestProcessingTime"])

	if err1 != nil || err2 != nil {
		return false
	} else {
		// longest time first
		return val1 > val2
	}
}
