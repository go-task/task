package task

// CheckCyclicDep checks if a task tree has any cyclic dependency
func (e *Executor) CheckCyclicDep() error {
	visits := make(map[string]struct{}, len(e.Tasks))

	var checkCyclicDep func(string, *Task) error
	checkCyclicDep = func(name string, t *Task) error {
		if _, ok := visits[name]; ok {
			return ErrCyclicDepDetected
		}
		visits[name] = struct{}{}
		defer delete(visits, name)

		for _, d := range t.Deps {
			// FIXME: ignoring by now. should return an error instead?
			task, ok := e.Tasks[d.Task]
			if !ok {
				continue
			}
			if err := checkCyclicDep(d.Task, task); err != nil {
				return err
			}
		}
		return nil
	}

	for k, v := range e.Tasks {
		if err := checkCyclicDep(k, v); err != nil {
			return err
		}
	}
	return nil
}
