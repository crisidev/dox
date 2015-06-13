package main

func analyzeStat(doxContainer *DoxContainer) {
	for _ = range doxContainer.c {
	}
}

func runInfluxStatPusher() {
	for _, doxContainer := range doxContainers {
		routineUp()
		go analyzeStat(doxContainer)
	}
}
