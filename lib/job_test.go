package jobworker

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartJob(t *testing.T) {
	fmt.Println("starting job")
	newJob, err := NewJob([]string{"ls"})
	assert.Nil(t, err, "error creating new job")

	go func() {
		newJob.Start()
	}()

	for out := range newJob.outputChan {
		fmt.Println(out)
	}
}

func TestStartJobWithArgument(t *testing.T) {
	fmt.Println("starting job")
	newJob, err := NewJob([]string{"ls", "-al"})
	assert.Nil(t, err, "error creating new job")

	go func() {
		newJob.Start()
	}()

	for out := range newJob.outputChan {
		fmt.Println(out)
	}
}

func TestLongRunningJobWithStop(t *testing.T) {
	newJob, err := NewJob([]string{"ping", "127.0.0.1"})
	assert.Nil(t, err, "error creating new job")
	go func() {
		newJob.Start()
	}()
	time.Sleep(1 * time.Second)
	assert.Equal(t, "running", newJob.Status())

	go func() {
		for out := range newJob.outputChan {
			fmt.Println(out)
		}
	}()

	time.Sleep(3 * time.Second)
	newJob.Stop()
	assert.Equal(t, "stopped", newJob.Status())
}

func TestQueryJob(t *testing.T) {
	newJob, err := NewJob([]string{"ping", "127.0.0.1"})
	assert.Nil(t, err, "error creating new job")

	go func() {
		newJob.Start()
	}()

	time.Sleep(1 * time.Second)

	assert.Equal(t, "running", newJob.Status())
	go func() {
		for out := range newJob.outputChan {
			fmt.Println(out)
		}
	}()
	newJob.Stop()
	assert.Equal(t, "stopped", newJob.Status())

}
