package main

import (
	"fmt"
	"github.com/rubyist/circuitbreaker"
	"math/rand"
	"time"
)

func main() {
	cb := circuit.NewRateBreaker(0.56, 20)

	events := cb.Subscribe()
	go func() {
		for {
			e := <-events
			switch e {
			case circuit.BreakerTripped:
				fmt.Println("event: TRIPPED")
			case circuit.BreakerReset:
				fmt.Println("event: RESET")
			case circuit.BreakerReady:
				fmt.Println("event: READY")
			case circuit.BreakerFail:
				fmt.Println("event: FAIL")
			}
		}
	}()

	go func() {
		for {
			fmt.Printf("failures: %d   successes: %d  rate: %f\n", cb.Failures(), cb.Successes(), cb.ErrorRate())
			time.Sleep(time.Second)
		}
	}()

	for {
		if cb.Ready() {
			if rand.Intn(2) == 1 {
				cb.Success()
			} else {
				cb.Fail()
			}
		} else {
			// fmt.Println("CB NOT READY")
		}

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(300)))
	}
}
