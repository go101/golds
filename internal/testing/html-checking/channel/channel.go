package channel

func f() {
	var c chan int
	select {
	case c <- 1:
	case <-c:
	case _ = <-c:
	case _, _ = <-c:
	}
}
