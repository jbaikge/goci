package main

import "log"

var (
	save_item = make(chan interface{})
)

func run_saver() {
	type ider interface {
		WholeID() string
		GetInfo() TaskInfo
	}
	var id ider
	for it := range save_item {
		var coll string
		switch it.(type) {
		case *Work:
			coll = "Work"
		case *Build:
			coll = "Build"
		case *Test:
			coll = "Test"
		default:
			log.Printf("don't know how to save an item of type %T", it)
			continue
		}
		_ = coll
		id = it.(ider)
		good := id.GetInfo().Error == ""
		log.Println(id.WholeID(), "save. good:", good)
		if !good {
			log.Printf("%s error: %q", id.WholeID(), id.GetInfo().Error)
		}
		if t, ok := it.(*Test); ok && good {
			log.Printf("%s passed: %v output: %q", id.WholeID(), t.Passed, t.Output)
		}
	}
}