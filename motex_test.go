package motex

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func TestMotex(t *testing.T) {
	var mo Motex
	var state int

	increment := func() {
		mo.Lock()
		defer mo.Unlock()
		state = state + 1
	}
	demotedIncrement := func() {
		mo.Lock()
		defer mo.Unlock()
		mo.Demote()
		old := state
		mo.Promote()
		state = old + 1
	}
	decrementAndDemotedIncrement := func() {
		mo.Lock()
		defer mo.Unlock()
		state = state - 1
		mo.Demote()
		old := state
		mo.Promote()
		state = old + 2
	}

	tests := []struct {
		name  string
		funcs []func()
	}{
		{
			name: "func1",
			funcs: []func(){
				increment,
			},
		},
		{
			name: "func2",
			funcs: []func(){
				demotedIncrement,
			},
		},
		{
			name: "func3",
			funcs: []func(){
				decrementAndDemotedIncrement,
			},
		},
		{
			name: "func1 AND func2",
			funcs: []func(){
				increment,
				demotedIncrement,
			},
		},
		{
			name: "func2 AND func3",
			funcs: []func(){
				demotedIncrement,
				decrementAndDemotedIncrement,
			},
		},
		{
			name: "func1 AND func3",
			funcs: []func(){
				increment,
				decrementAndDemotedIncrement,
			},
		},
		{
			name: "func1 AND func2 AND func3",
			funcs: []func(){
				increment,
				demotedIncrement,
				decrementAndDemotedIncrement,
			},
		},
	}

	var n int
	if RACE {
		n = 2700 // Go race detector has a limit of 8192 goroutines. 2700 < 8192 / max(len(test.funcs))
	} else {
		n = 100000
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state = 0
			c := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(n * len(test.funcs))
			for i := 0; i < n; i++ {
				for _, f := range test.funcs {
					go func() {
						<-c
						f()
						wg.Done()
					}()
				}
			}
			close(c)
			wg.Wait()
			want := n * len(test.funcs)
			if state != want {
				t.Errorf("state=%v, want %v", state, want)
			}
		})
	}
}

func TestIncorrectMotexUse(t *testing.T) {
	var mo Motex
	var state int

	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "non-atomic read (rlock) and write (lock)",
			f: func() {
				mo.RLock()
				old := state
				mo.RUnlock()

				mo.Lock()
				state = old + 1
				mo.Unlock()
			},
		},
		{
			name: "non-atomic read (lock) and write (lock)",
			f: func() {
				mo.Lock()
				old := state
				mo.Unlock()

				mo.Lock()
				state = old + 1
				mo.Unlock()
			},
		},
		{
			name: "non-atomic read (demoted) and write (lock)",
			f: func() {
				mo.Lock()
				mo.Demote()
				old := state
				mo.Promote()
				mo.Unlock()

				mo.Lock()
				state = old + 1
				mo.Unlock()
			},
		},
	}

	var n int
	if RACE {
		n = 8000 // Go race detector has a limit of 8192 goroutines.
	} else {
		n = 100000
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state = 0
			c := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(n)
			for i := 0; i < n; i++ {
				go func() {
					<-c
					test.f()
					wg.Done()
				}()
			}
			close(c)
			wg.Wait()
			if state >= n {
				t.Errorf("state=%v, want state < %v", state, n)
			}
		})
	}
}

// TestMotexMisuse and init copied from the std library's mutex tests in the
// sync package.
func init() {
	if len(os.Args) == 3 && os.Args[1] == "TESTMISUSE" {
		for _, test := range misuseTests {
			if test.name != os.Args[2] {
				continue
			}
			test.f()
			fmt.Printf("test completed\n")
			os.Exit(0)
		}
		fmt.Printf("unknown test\n")
		os.Exit(0)
	}
}

var misuseTests = []struct {
	name string
	f    func()
}{
	{
		name: "Unlock without Lock",
		f: func() {
			var mo Motex
			mo.Unlock()
		},
	},
	{
		name: "2 Unlocks after Lock",
		f: func() {
			var mo Motex
			mo.Lock()
			mo.Unlock()
			mo.Unlock()
		},
	},
	{
		name: "RUnlock without RLock",
		f: func() {
			var mo Motex
			mo.RUnlock()
		},
	},
	{
		name: "2 RUnlocks after RLock",
		f: func() {
			var mo Motex
			mo.RLock()
			mo.RUnlock()
			mo.RUnlock()
		},
	},
	{
		name: "Promote outside of Lock",
		f: func() {
			var mo Motex
			mo.Promote()
		},
	},
	{
		name: "Demote outside of Lock",
		f: func() {
			var mo Motex
			mo.Demote()
		},
	},
	{
		name: "Promote without Demote",
		f: func() {
			var mo Motex
			mo.Lock()
			mo.Promote()
		},
	},
	{
		name: "2 Demotes in Lock",
		f: func() {
			var mo Motex
			mo.Lock()
			mo.Demote()
			mo.Demote()
		},
	},
	{
		name: "2 Promotes after Demote",
		f: func() {
			var mo Motex
			mo.Lock()
			mo.Demote()
			mo.Promote()
			mo.Promote()
		},
	},
	{
		name: "Unlock before Demote",
		f: func() {
			var mo Motex
			mo.Lock()
			mo.Unlock()
			mo.Demote()
		},
	},
	{
		name: "Unlock before Promote",
		f: func() {
			var mo Motex
			mo.Lock()
			mo.Demote()
			mo.Unlock()
		},
	},
	{
		name: "RUnlock after Lock",
		f: func() {
			var mo Motex
			mo.Lock()
			mo.RUnlock()
		},
	},
	{
		name: "Unlock after RLock",
		f: func() {
			var mo Motex
			mo.RLock()
			mo.Unlock()
		},
	},
}

func TestMotexMisuse(t *testing.T) {
	for _, test := range misuseTests {
		out, err := exec.Command(os.Args[0], "TESTMISUSE", test.name).CombinedOutput()
		if err == nil || !strings.Contains(string(out), "unlocked") {
			t.Errorf("%s: did not find failure with message about unlocked lock: %s\n%s\n", test.name, err, out)
		}
	}
}

// A failure for this test will look like a deadlock (or timeout / hang if -race is enabled).
func TestMotexDoesntDeadlock(_ *testing.T) {
	var mo Motex

	mo.Lock()
	mo.Demote()
	mo.Promote()
	mo.Demote()
	mo.RLock()
	mo.RLock()
	mo.RUnlock()
	mo.RUnlock()
	mo.Promote()
	mo.Unlock()

	mo.RLock()
	mo.RLock()
	mo.RUnlock()
	mo.RUnlock()
}
