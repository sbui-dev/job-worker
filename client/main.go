package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"os"

	worker "github.com/sbui-dev/jobworker/data/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/alecthomas/kingpin/v2"
)

const (
	certFolder     = "../data/certs/"
	serverCAPath   = "server_ca_cert.pem"
	clientCertPath = "alice_cert.pem"
	clientKeyPath  = "alice_key.pem"
	serverAddress  = "localhost:50005"
)

var (
	app  = kingpin.New("jobclient", "A command-line job client.")
	addr = app.Flag("addr", "The address to connect to").Default(serverAddress).String()
	user = app.Flag("user", "Name of user: alice, bob, carl").Default("alice").String()

	start = app.Command("start", "Start a job")
	cmd   = start.Arg("command", "command to run").Required().Strings()

	stop   = app.Command("stop", "Stop a job")
	stopid = stop.Arg("id", "job id").Required().String()

	query   = app.Command("query", "Query a job")
	queryid = query.Arg("id", "job id").Required().String()

	logCmd = app.Command("log", "Get job logs")
	logid  = logCmd.Arg("id", "job id").Required().String()
)

func startJob(client worker.WorkerClient, message []string) {
	fmt.Println("sending job")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp, err := client.JobStart(ctx, &worker.WorkerStartRequest{Command: message})
	if err != nil {
		log.Fatalf("client = %v: ", err)
	}

	jobLog(client, resp.JobId)
}

func stopJob(client worker.WorkerClient, jobID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := client.JobStop(ctx, &worker.WorkerStopRequest{JobId: jobID})
	if err != nil {
		log.Fatalf("client = %v: ", err)
	}
}

func queryJob(client worker.WorkerClient, jobID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp, err := client.JobQuery(ctx, &worker.WorkerQueryRequest{JobId: jobID})
	if err != nil {
		log.Fatalf("client = %v: ", err)
	}
	log.Printf("Job status is: %s", resp.Status)
}

func jobLog(client worker.WorkerClient, jobID string) {
	fmt.Println("sending log request")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp, err := client.JobLog(ctx, &worker.WorkerLogRequest{JobId: jobID})
	if err != nil {
		log.Fatalf("client = %v: ", err)
	}
	done := make(chan struct{})

	go func() {
		for {
			resp, err := resp.Recv()
			if err == io.EOF {
				done <- struct{}{} //means stream is finished
				return
			}
			if err != nil {
				log.Fatalf("cannot receive %v", err)
			}
			log.Printf("[%s]: %s", resp.JobId, resp.Log)
		}
	}()

	<-done
}

func setupTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFolder+clientCertPath, certFolder+clientKeyPath)
	if err != nil {
		// todo fix errors
		log.Fatalf("failed to load client cert: %v", err)
	}

	ca := x509.NewCertPool()
	caBytes, err := os.ReadFile(certFolder + serverCAPath)
	if err != nil {
		log.Fatalf("failed to read ca cert %q: %v", serverCAPath, err)
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		log.Fatalf("failed to parse %q", serverCAPath)
	}

	tlsConfig := &tls.Config{
		ServerName:   "job-server",
		Certificates: []tls.Certificate{cert},
		RootCAs:      ca,
	}
	return tlsConfig, nil
}

func main() {
	fmt.Println("setting up tls")
	tlsConfig, err := setupTLSConfig()
	if err != nil {
		log.Fatalf("failed to setup tls config %v", err)
	}

	serverAddr := fmt.Sprintf("passthrough:///%s", serverAddress)
	if *addr != "" {
		serverAddr = fmt.Sprintf("passthrough:///%s", *addr)
	}
	fmt.Printf("setting up conn with %s\n", serverAddr)
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("setting up client")
	workerClient := worker.NewWorkerClient(conn)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case start.FullCommand():
		startJob(workerClient, *cmd)
	case stop.FullCommand():
		stopJob(workerClient, *stopid)
	case query.FullCommand():
		queryJob(workerClient, *queryid)
	case logCmd.FullCommand():
		jobLog(workerClient, *logid)
	}
}
