package task

func HasCyclicDep(m map[string]*Task) bool {
	visits := make(map[string]struct{}, len(m))

	var checkCyclicDep func(string, *Task) bool
	checkCyclicDep = func(name string, t *Task) bool {
		if _, ok := visits[name]; ok {
			return false
		}
		visits[name] = struct{}{}
		defer delete(visits, name)

		for _, d := range t.Deps {
			if !checkCyclicDep(d, m[d]) {
				return false
			}
		}
		return true
	}

	for k, v := range m {
		if !checkCyclicDep(k, v) {
			return true
		}
	}
	return false
}
