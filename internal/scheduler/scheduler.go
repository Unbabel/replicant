package scheduler

/*
   Copyright 2019 Bruno Moura <brunotm@gmail.com>

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"context"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	ErrorTaskAlreadyExists = fmt.Errorf("task already exists")
	ErrorNoSuchTask        = fmt.Errorf("no such task")
)

// Task represents a test task to be scheduled
type Task interface {
	Run()
}

// TaskFunc is a scheduler task
type TaskFunc func()

// Run implements Task interface
func (t TaskFunc) Run() { t() }

// Entry is a scheduled entry
type Entry struct {
	Name     string
	ID       int
	Schedule string
}

// Scheduler for transactions
type Scheduler struct {
	mtx   sync.Mutex
	cron  *cron.Cron
	tasks map[string]Entry
}

// New creates a new scheduler
func New() (scheduler *Scheduler) {

	scheduler = &Scheduler{}
	scheduler.tasks = make(map[string]Entry)

	scheduler.cron = cron.New(
		cron.WithLogger(cron.DefaultLogger),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
			cron.Recover(cron.DefaultLogger),
		),
	)

	return scheduler
}

// Start the scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop the scheduler. The returned context can be used to wait for all
// running jobs to finish.
func (s *Scheduler) Stop() (done context.Context) {
	return s.cron.Stop()
}

// AddTask to the scheduler
func (s *Scheduler) AddTask(name, schedule string, task Task) (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.tasks[name]; ok {
		return ErrorTaskAlreadyExists
	}

	var id cron.EntryID
	if id, err = s.cron.AddJob(schedule, task); err != nil {
		return fmt.Errorf("scheduler: error adding job: %w", err)
	}

	s.tasks[name] = Entry{Name: name, ID: int(id), Schedule: schedule}

	return nil
}

// AddTaskFunc is like AddTask but accepts a function task
func (s *Scheduler) AddTaskFunc(name, schedule string, task func()) (err error) {
	return s.AddTask(name, schedule, TaskFunc(task))
}

// RemoveTask from the scheduler
func (s *Scheduler) RemoveTask(name string) (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	var entry Entry
	var ok bool
	if entry, ok = s.tasks[name]; !ok {
		return ErrorNoSuchTask
	}

	s.cron.Remove(cron.EntryID(entry.ID))
	delete(s.tasks, name)

	return nil
}

// Entries returns all current tasks from the scheduler
func (s *Scheduler) Entries() (entries []Entry) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	entries = make([]Entry, 0, len(s.tasks))
	for _, e := range s.tasks {
		entries = append(entries, e)
	}

	return entries
}
