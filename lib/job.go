// Copyright 2023 Steven Bui

package jobworker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/google/uuid"
)

const (
	RunningStatus = "running"
	StoppedStatus = "stopped"
)

type JobInfo struct {
	JobID      string
	status     string
	cancelJob  context.CancelFunc
	outputChan chan string
	command    []string
	output     *bytes.Buffer
}

func NewJob(command []string) (*JobInfo, error) {
	jobID := uuid.New().String()
	outChannel := make(chan string)
	outBuf := bytes.NewBuffer([]byte{})

	job := JobInfo{
		JobID:      jobID,
		command:    command,
		outputChan: outChannel,
		output:     outBuf,
	}

	return &job, nil
}

// execute() helper func to execute command
func (j *JobInfo) execute(ctx context.Context) error {
	log.Printf("executing")
	cmd := exec.CommandContext(ctx, j.command[0], j.command[1:]...)
	//cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	out, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Errorf("%v", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		fmt.Errorf("%v", err)
		return err
	}
	fmt.Println("execute completed")

	go func() {
		defer close(j.outputChan)
		scan := bufio.NewScanner(out)
		for scan.Scan() {
			m := scan.Text()
			j.output.WriteString(fmt.Sprintf("%s\n", m))
			j.outputChan <- m
		}
	}()

	cmd.Wait()
	fmt.Println("RETURNING: execute wait done")
	return nil
}

// Start - starts a job
func (jw *JobInfo) Start() {
	log.Printf("Start job")
	ctx, cancel := context.WithCancel(context.Background())
	jw.cancelJob = cancel
	jw.status = RunningStatus
	fmt.Printf("status is %s\n", jw.status)
	err := jw.execute(ctx)
	if err != nil {
		fmt.Printf("error encountered in job")
		return
	}
	fmt.Println("start execute done")
	jw.status = StoppedStatus
	fmt.Printf("status is %s\n", jw.status)
	fmt.Println("start job done")

	return
}

// Stop - stop a job
func (jw *JobInfo) Stop() {
	jw.status = StoppedStatus
	jw.cancelJob()
	jw.outputChan <- "Job has been stopped by user\n"
	return
}

// todo rename to query
func (jw *JobInfo) Status() string {
	return jw.status
}

func (jw *JobInfo) IsRunning() bool {
	if jw.status == RunningStatus {
		return true
	}
	return false
}

func (jw *JobInfo) GetLog() <-chan string {
	fmt.Println("Get Log")
	outChan := make(chan string)
	reader := bufio.NewReader(jw.output)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("returning out chan")
			close(outChan)
			return outChan
		}
		outChan <- line
	}
}

func (jw *JobInfo) GetOutputChannel() <-chan string {
	fmt.Println("sending output log")
	return jw.outputChan
}
