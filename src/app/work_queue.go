package main

import (
	"builder"
	"fmt"
	"log"
	"time"
)

const queue_size = 100

var work_queue = make(chan builder.Work, queue_size)

type TaskInfo struct {
	When  time.Time
	ID    string
	Error string
}

func (t TaskInfo) GetInfo() TaskInfo {
	return t
}

type Work struct {
	TaskInfo
	Work builder.Work
}

func (w *Work) WholeID() string {
	return w.ID
}

type Build struct {
	TaskInfo
	WorkID string
	Build  builder.Build

	poke chan bool
}

func (b *Build) cleanup(num int) {
	defer log.Println(b.WholeID(), "clean up")
	defer b.Build.Cleanup()

	for i := 0; i < num; i++ {
		_, ok := <-b.poke
		if !ok {
			return
		}
	}
}

func (b *Build) WholeID() string {
	return fmt.Sprintf("%s:%s", b.WorkID, b.ID)
}

type Test struct {
	TaskInfo
	WorkID  string
	BuildID string
	Path    string

	Output   string
	Passed   bool
	Started  time.Time
	Duration time.Duration

	done chan bool
}

func (t *Test) Start() {
	if t.Started.IsZero() {
		t.Started = time.Now()
	}
}

func (t *Test) Finish() {
	t.Duration = time.Since(t.Started)
	t.done <- true
}

func (t *Test) WholeID() string {
	return fmt.Sprintf("%s:%s:%s", t.WorkID, t.BuildID, t.ID)
}

func new_info() (t TaskInfo) {
	t.When = time.Now()
	t.ID = new_id()
	return
}

func new_test(path string, build *Build, work *Work) (t *Test) {
	t = &Test{
		TaskInfo: new_info(),
		Path:     path,
		BuildID:  build.ID,
		WorkID:   work.ID,

		done: build.poke,
	}

	return
}

func new_build(build builder.Build, work *Work) (b *Build) {
	b = &Build{
		TaskInfo: new_info(),
		Build:    build,
		WorkID:   work.ID,

		poke: make(chan bool),
	}
	return
}

func new_work(work builder.Work) (w *Work) {
	w = &Work{
		TaskInfo: new_info(),
		Work:     work,
	}
	return
}

func run_work_queue() {
	for work := range work_queue {
		log.Println("got work item:", work.RepoPath())
		w := new_work(work)

		//create the builds for the work
		builds, err := builder.CreateBuilds(work)
		if err != nil {
			w.Error = err.Error()
			save_item <- w
			continue
		}
		save_item <- w

		//build the work struct out to include all the tests
		for _, build := range builds {
			b := new_build(build, w)
			if err := build.Error(); err != nil {
				b.Error = err.Error()
				b.cleanup(0)
				save_item <- b
				continue
			}
			save_item <- b

			paths := build.Paths()
			for _, path := range paths {
				t := new_test(path, b, w)
				schedule_test <- t
			}

			//launch a goroutine to handle cleanup after x tests have run
			go b.cleanup(len(paths))
		}
	}
}