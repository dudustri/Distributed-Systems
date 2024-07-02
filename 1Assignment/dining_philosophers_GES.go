package main

import (
	"fmt"
	"math/rand"
	"time"
)

const numPhilosophers = 5 // number of philosophers

type Philosopher struct {
	ID           int		// ID of the philosopher
	LeftFork     chan bool 	// creating channels of type bool;; bidirectional channel type. Compilers allow both receiving values and sending values to channels
	RightFork    chan bool 	// channels for acquiring and releasing forks
}

func (p *Philosopher) dine() {
	for i := 0; i < 3; i++ {
		// To prevent deadlock, one philosopher tries to pick the right fork first
		if p.ID == numPhilosophers-1 {
			p.pickUp(p.RightFork, p.LeftFork) // last philosopher is picking the forks in the opposite order (breaking the symmetry)
		} else {
			p.pickUp(p.LeftFork, p.RightFork)  // other philophers pic up left first then right
		}
		p.eat()
		p.think()
		p.putDown()
	}
	fmt.Printf("Philosopher %d has finished eating 3 times.\n", p.ID)
}

func (p *Philosopher) think() {
	fmt.Printf("Philosopher %d is thinking\n", p.ID)
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
}

func (p *Philosopher) eat() {
	fmt.Printf("Philosopher %d is eating\n", p.ID)
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
}

func (p *Philosopher) pickUp(firstFork, secondFork chan bool) {
	<-firstFork
	<-secondFork
}

func (p *Philosopher) putDown() {
	p.LeftFork <- true
	p.RightFork <- true
}

func forkManager(fork chan bool) {
	/*
	manages a single fork 
	:param channel named fork of type fork - represents the fork - channel of fork is size 1
	*/ 

	// keep on repeating indefinitely
    for {
		fork <- true // sends (or writes) a true value into the fork;;  represents that the fork is available on the table
		<-fork // The function waits to receive (or read) a value from the fork;; put the fork back on the table after using it.
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Initialize forks
	forks := make([]chan bool, numPhilosophers) // slice of channels of type bool with 5 bools
	for i := 0; i < numPhilosophers; i++ {
		forks[i] = make(chan bool, 1)  // buffer size of 1 - Initialize each channel in the slice - create a channel if forks (a channel of size 1) and add to forks 
		go forkManager(forks[i])
	}

	// Start philosophers
	for i := 0; i < numPhilosophers; i++ {
		p := &Philosopher{
			ID:        i,
			LeftFork:  forks[i],
			RightFork: forks[(i+1)%numPhilosophers],
		}
		go p.dine()
	}

	// Allow philosophers some time to dine
	time.Sleep(20 * time.Second)
}
