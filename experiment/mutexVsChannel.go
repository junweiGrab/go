package experiment

import "sync"

var mutex = sync.Mutex{}
var ch = make(chan bool, 1)

func UseMutex(i int) {
	mutex.Lock()
	// fmt.Printf("m#%v", i)
	mutex.Unlock()
}

func UseChannel(i int) {
	ch <- true
	// fmt.Printf("c#%v", i)
	<-ch
}
