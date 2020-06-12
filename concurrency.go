package task

func (e *Executor) acquireConcurrencyLimit() func() {
	if e.concurrencySemaphore == nil {
		return emptyFunc
	}

	e.concurrencySemaphore <- struct{}{}
	return func() {
		<-e.concurrencySemaphore
	}
}

func (e *Executor) releaseConcurrencyLimit() func() {
	if e.concurrencySemaphore == nil {
		return emptyFunc
	}

	<-e.concurrencySemaphore
	return func() {
		e.concurrencySemaphore <- struct{}{}
	}
}

func emptyFunc() {}
